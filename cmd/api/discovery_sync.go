package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/parakliite/simila/internal/data"
)

const (
	discoverySyncInterval = 6 * time.Hour
	discoverySyncMaxPages = int32(50)
	discoverySyncMaxFails = int32(5)
	discoverySyncPause    = 24 * time.Hour
)

var errDiscoverySyncPaused = errors.New("discovery sync paused after consecutive failures")

type discoverySyncResult struct {
	ID         int32   `json:"id"`
	Popularity float64 `json:"popularity"`
}

type discoverySyncResponse struct {
	Page       int32                 `json:"page"`
	TotalPages int32                 `json:"total_pages"`
	Results    []discoverySyncResult `json:"results"`
}

func (app *application) discoverySyncJob(ctx context.Context) {
	ticker := time.NewTicker(discoverySyncInterval)
	defer ticker.Stop()

	mediaType := "tv"
	for {
		select {
		case <-ctx.Done():
			app.logger.PrintInfo("discovery sync stopped by context cancellation", nil)
			return
		case <-ticker.C:
			if err := app.syncDiscoveryPage(ctx, mediaType); err != nil && !errors.Is(err, errDiscoverySyncPaused) {
				app.logger.PrintError(err, map[string]string{
					"worker":    "discovery sync",
					"mediaType": mediaType,
				})
			}
			if mediaType == "tv" {
				mediaType = "movie"
			} else {
				mediaType = "tv"
			}
		}
	}
}

func (app *application) syncDiscoveryPage(ctx context.Context, mediaType string) error {
	paused, err := app.discoverySyncPaused(ctx)
	if err != nil {
		return err
	}
	if paused {
		return nil
	}

	state, err := app.models.Discovery.GetDiscoverySyncState(ctx, mediaType)
	if err != nil {
		return fmt.Errorf("get %s discovery sync state: %w", mediaType, err)
	}

	response, err := app.fetchDiscoveryPage(ctx, mediaType, state.NextPage)
	if err != nil {
		if recordErr := app.recordDiscoverySyncFailure(ctx, mediaType, state.ConsecutiveFailures); recordErr != nil {
			return recordErr
		}
		return err
	}

	consecutiveFailures := state.ConsecutiveFailures
	for _, result := range response.Results {
		err := app.syncDiscoveryResult(ctx, mediaType, result)
		if err != nil {
			consecutiveFailures++
			if recordErr := app.recordDiscoverySyncFailure(ctx, mediaType, consecutiveFailures-1); recordErr != nil {
				return recordErr
			}
			continue
		}

		if consecutiveFailures > 0 {
			if err := app.models.Discovery.UpdateDiscoverySyncFailure(ctx, mediaType, 0, nil); err != nil {
				return fmt.Errorf("reset %s discovery sync failures: %w", mediaType, err)
			}
		}
		consecutiveFailures = 0
	}

	nextPage := state.NextPage + 1
	lastPage := response.TotalPages
	if lastPage <= 0 || lastPage > discoverySyncMaxPages {
		lastPage = discoverySyncMaxPages
	}
	if nextPage > lastPage {
		nextPage = 1
	}

	if err := app.models.Discovery.UpdateDiscoverySyncSuccess(ctx, mediaType, nextPage); err != nil {
		return fmt.Errorf("save %s discovery sync checkpoint: %w", mediaType, err)
	}

	return nil
}

func (app *application) discoverySyncPaused(ctx context.Context) (bool, error) {
	for _, mediaType := range []string{"tv", "movie"} {
		state, err := app.models.Discovery.GetDiscoverySyncState(ctx, mediaType)
		if err != nil {
			return false, err
		}
		if state.PausedUntil != nil && time.Now().Before(*state.PausedUntil) {
			return true, nil
		}
	}
	return false, nil
}

func (app *application) fetchDiscoveryPage(
	ctx context.Context,
	mediaType string,
	page int32,
) (discoverySyncResponse, error) {
	query := url.Values{
		"language":                     []string{"en-US"},
		"include_adult":                []string{"false"},
		"include_null_first_air_dates": []string{"false"},
		"sort_by":                      []string{"popularity.desc"},
		"page":                         []string{strconv.FormatInt(int64(page), 10)},
	}

	req, err := app.newTMDBRequest(ctx, []string{"3", "discover", mediaType}, query)
	if err != nil {
		return discoverySyncResponse{}, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	// #nosec G704 -- TMDB URL is built from a validated base URL and fixed path segments.
	resp, err := client.Do(req)
	if err != nil {
		return discoverySyncResponse{}, fmt.Errorf("fetch %s discovery page: %w", mediaType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return discoverySyncResponse{}, fmt.Errorf("fetch %s discovery page: status code %d", mediaType, resp.StatusCode)
	}

	var result discoverySyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return discoverySyncResponse{}, fmt.Errorf("decode %s discovery page: %w", mediaType, err)
	}

	return result, nil
}

func (app *application) syncDiscoveryResult(
	ctx context.Context,
	mediaType string,
	result discoverySyncResult,
) error {
	_, err := app.models.Movies.GetMedia(ctx, result.ID, mediaType)
	if err == nil {
		return app.models.Movies.UpdateMediaPopularity(ctx, result.ID, mediaType, result.Popularity)
	}
	if !errors.Is(err, data.ErrRecordNotFound) {
		return err
	}

	media, err := app.fetchTMDBMedia(ctx, result.ID, mediaType)
	if err != nil {
		return err
	}
	media.Popularity = result.Popularity

	if err := app.models.Movies.InsertMedia(ctx, media); err != nil {
		return fmt.Errorf("insert %s media %d: %w", mediaType, result.ID, err)
	}

	return nil
}

func (app *application) recordDiscoverySyncFailure(
	ctx context.Context,
	mediaType string,
	previousFailures int32,
) error {
	failures := previousFailures + 1
	var pausedUntil *time.Time
	if failures >= discoverySyncMaxFails {
		paused := time.Now().Add(discoverySyncPause)
		pausedUntil = &paused
	}

	if err := app.models.Discovery.UpdateDiscoverySyncFailure(ctx, mediaType, failures, pausedUntil); err != nil {
		return fmt.Errorf("record %s discovery sync failure: %w", mediaType, err)
	}
	if pausedUntil != nil {
		return errDiscoverySyncPaused
	}
	return nil
}
