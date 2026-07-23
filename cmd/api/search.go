package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"golang.org/x/sync/errgroup"
)

func (app *application) searchHandler(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query().Get("query")
	if query == "" {
		app.badRequestResponse(w, req, errors.New("query parameter is required"))
		return
	}

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 10)

	var users []data.User
	var media_results []TMDBSearchResult

	g, ctx := errgroup.WithContext(req.Context())

	g.Go(func() error {
		var err error
		media_results, err = app.mediaSearch(query)
		return err
	})

	g.Go(func() error {
		var err error
		users, err = app.userSearch(ctx, query, int32(limit))
		return err
	})

	if err := g.Wait(); err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"media_results": media_results, "users_results": users}, nil)
}

func (app *application) userSearch(ctx context.Context, query string, limit int32) ([]data.User, error) {
	users, err := app.models.Users.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (app *application) mediaSearch(query string) ([]TMDBSearchResult, error) {
	url := fmt.Sprintf("%s/3/search/multi?query=%s", app.config.tmdbBaseURL, url.QueryEscape(query))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+app.config.tmdbToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return nil, errors.New("failed to search TMDB")
	}
	defer resp.Body.Close()

	var searchResults struct {
		Results []TMDBSearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		return nil, err
	}

	results := make([]TMDBSearchResult, 0, 20)
	for _, result := range searchResults.Results {
		if result.MediaType == "movie" || result.MediaType == "tv" {
			results = append(results, result)
		}
	}

	return results, nil
}
