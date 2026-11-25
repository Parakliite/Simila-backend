package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/parakliite/simila/internal/data"
)

// This is the response from tmdb and not the application itself
type UnifiedTMDBResponse struct {
	ID int32 `json:"id"`
	// Movie fields
	Title         string `json:"title"`
	OriginalTitle string `json:"original_title"`
	ReleaseDate   string `json:"release_date"`
	Runtime       int32  `json:"runtime"`
	// TV fields
	Name         string  `json:"name"`
	OriginalName string  `json:"original_name"`
	FirstAirDate string  `json:"first_air_date"`
	EpisodeRun   []int32 `json:"episode_run_time"`
	LastEpisode  *struct {
		Runtime int32 `json:"runtime"`
	} `json:"last_episode_to_air"`
	// Shared fields
	Overview     string `json:"overview"`
	PosterPath   string `json:"poster_path"`
	BackdropPath string `json:"backdrop_path"`
	Genres       []struct {
		Name string `json:"name"`
	} `json:"genres"`
}

type TMDBSearchResult struct {
	BackdropPath  string `json:"backdrop_path"`
	ID            int    `json:"id"`
	Title         string `json:"title,omitempty"`
	OriginalTitle string `json:"original_title,omitempty"`
	PosterPath    string `json:"poster_path"`
	MediaType     string `json:"media_type"`
	ReleaseDate   string `json:"release_date,omitempty"`
	Name          string `json:"name,omitempty"`
	OriginalName  string `json:"original_name,omitempty"`
	FirstAirDate  string `json:"first_air_date,omitempty"`
}

type mediaCastMember struct {
	Name        string  `json:"name"`
	ProfilePath *string `json:"profile_path"`
	Character   string  `json:"character"`
}

// TODO: change this to use an anonymous/scoped struct instead
// of the returned value from the db query method
func (app *application) getMediaHandler(w http.ResponseWriter, req *http.Request) {
	id, err := app.readIDParam(req)
	if err != nil {
		app.notFoundResponse(w, req)
		return
	}

	mediaType := req.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, req, errors.New("type parameter must be 'movie' or 'tv'"))
		return
	}

	media, err := app.models.Movies.GetMedia(req.Context(), id, mediaType)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			media, err = app.fetchAndSaveMedia(req.Context(), id, mediaType)
			if err != nil {
				app.serverErrorResponse(w, req, err)
				return
			}
		} else {
			app.serverErrorResponse(w, req, err)
			return
		}
	}

	if media.Genre == nil {
		media.Genre = []data.Genre{}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"media": media}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) getMediaVideo(w http.ResponseWriter, req *http.Request) {
	id, err := app.readIDParam(req)
	if err != nil {
		app.notFoundResponse(w, req)
		return
	}

	mediaType := req.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, req, errors.New("type parameter must be 'movie' or 'tv'"))
		return
	}

	path := []string{"3", mediaType, strconv.FormatInt(int64(id), 10), "videos"}

	tmdbReq, err := app.newTMDBRequest(req.Context(), path, url.Values{})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	// #nosec G704 -- TMDB URL is built from a validated base URL, fixed path segments, and a numeric ID.
	resp, err := client.Do(tmdbReq)
	if err != nil {
		app.logger.PrintError(err, map[string]string{"tmdb request": "request for videos"})
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}

	if resp.StatusCode != 200 {
		b := make([]byte, 1024)
		n, _ := resp.Body.Read(b)
		app.logger.PrintError(fmt.Errorf("failed to fetch from TMDB: statusCode: %v resp: %v", resp.StatusCode, string(b[:n])), nil)
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}
	defer resp.Body.Close()

	type TMDBVideoResponse struct {
		ID      int `json:"id"`
		Results []struct {
			Name string `json:"name"`
			Key  string `json:"key"`
			Site string `json:"site"`
			Type string `json:"type"`
		} `json:"results"`
	}

	var tmdbVidResp TMDBVideoResponse

	if err := json.NewDecoder(resp.Body).Decode(&tmdbVidResp); err != nil {
		app.logger.PrintError(err, map[string]string{"tmdb response": "was unable to get tmdb response"})
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}

	type Response struct {
		External string `json:"external"`
		Embed    string `json:"embed"`
	}

	var vidResp Response

	for _, result := range tmdbVidResp.Results {
		if strings.ToLower(result.Type) != "trailer" {
			continue
		}

		watch, embed, ok := app.videoURLs(result.Site, result.Key)
		if !ok {
			continue
		}

		vidResp.External, vidResp.Embed = watch, embed
		break
	}

	if err = app.writeJSON(w, http.StatusOK, envelope{"trailer": vidResp}, nil); err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) getMediaCast(w http.ResponseWriter, req *http.Request) {
	id, err := app.readIDParam(req)
	if err != nil {
		app.notFoundResponse(w, req)
		return
	}

	mediaType := req.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, req, errors.New("type parameter must be 'movie' or 'tv'"))
		return
	}

	path := []string{"3", mediaType, strconv.FormatInt(int64(id), 10), "credits"}
	tmdbReq, err := app.newTMDBRequest(req.Context(), path, url.Values{})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	// #nosec G704 -- TMDB URL is built from a validated base URL, fixed path segments, and a numeric ID.
	resp, err := client.Do(tmdbReq)
	if err != nil {
		app.logger.PrintError(err, map[string]string{"tmdb request": "request for credits"})
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b := make([]byte, 1024)
		n, _ := resp.Body.Read(b)
		app.logger.PrintError(fmt.Errorf("failed to fetch credits from TMDB: statusCode: %v resp: %v", resp.StatusCode, string(b[:n])), nil)
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}

	var tmdbResponse struct {
		Cast []mediaCastMember `json:"cast"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tmdbResponse); err != nil {
		app.logger.PrintError(err, map[string]string{"tmdb response": "unable to decode credits"})
		app.serverErrorResponse(w, req, fmt.Errorf("failed to fetch"))
		return
	}

	castCount := len(tmdbResponse.Cast)
	if castCount > 15 {
		castCount = 15
	}
	cast := append([]mediaCastMember(nil), tmdbResponse.Cast[:castCount]...)
	tmdbResponse.Cast = nil

	if err := app.writeJSON(w, http.StatusOK, envelope{"cast": cast}, nil); err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) fetchAndSaveMedia(
	ctx context.Context,
	tmdbID int32,
	mediaType string,
) (*data.Media, error) {
	newMedia, err := app.fetchTMDBMedia(ctx, tmdbID, mediaType)
	if err != nil {
		return nil, err
	}

	err = app.models.Movies.InsertMedia(ctx, newMedia)
	if err != nil {
		return nil, err
	}

	return newMedia, nil
}

func (app *application) fetchTMDBMedia(
	ctx context.Context,
	tmdbID int32,
	mediaType string,
) (*data.Media, error) {
	tmdbIDPath := strconv.FormatInt(int64(tmdbID), 10)
	var path []string
	switch mediaType {
	case "movie":
		path = []string{"3", "movie", tmdbIDPath}
	case "tv":
		path = []string{"3", "tv", tmdbIDPath}
	default:
		panic(fmt.Sprintf("an unexpected media type has been passed: %s", mediaType))
	}

	req, err := app.newTMDBRequest(ctx, path, url.Values{})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	// #nosec G704 -- TMDB URL is built from a validated base URL, fixed path segments, and a numeric ID.
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from TMDB: %v", err)
	}

	if resp.StatusCode != 200 {
		b := make([]byte, 1024)
		n, _ := resp.Body.Read(b)
		return nil, fmt.Errorf("failed to fetch from TMDB: statusCode: %v resp: %v", resp.StatusCode, string(b[:n]))
	}
	defer resp.Body.Close()

	var r UnifiedTMDBResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, err
	}

	newMedia := &data.Media{
		TmdbID:       r.ID,
		MediaType:    mediaType,
		Overview:     r.Overview,
		PosterPath:   r.PosterPath,
		BackdropPath: r.BackdropPath,
		Genre:        []data.Genre{},
	}

	if mediaType == "movie" {
		newMedia.Title = r.Title
		newMedia.OriginalTitle = r.OriginalTitle
		newMedia.Runtime = data.Runtime(r.Runtime)
		if newMedia.Runtime == 0 {
			newMedia.Runtime = data.Runtime(1)
		}
		newMedia.ReleaseDate, _ = time.Parse("2006-01-02", r.ReleaseDate)
	} else {
		newMedia.Title = r.Name
		newMedia.OriginalTitle = r.OriginalName
		newMedia.ReleaseDate, _ = time.Parse("2006-01-02", r.FirstAirDate)
		if len(r.EpisodeRun) > 0 {
			newMedia.Runtime = data.Runtime(r.EpisodeRun[0])
		}
		if newMedia.Runtime == 0 && r.LastEpisode != nil {
			newMedia.Runtime = data.Runtime(r.LastEpisode.Runtime)
		}
		if newMedia.Runtime == 0 {
			newMedia.Runtime = data.Runtime(1)
		}
	}

	for _, g := range r.Genres {
		newMedia.Genre = append(newMedia.Genre, data.Genre{GenreName: g.Name})
	}

	return newMedia, nil
}
