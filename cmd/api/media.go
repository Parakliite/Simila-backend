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

// TODO: change this to use an anonymous/scoped struct instead
// of the returned value from the db query method
func (app *application) getMediaHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	mediaType := r.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, r, errors.New("type parameter must be 'movie' or 'tv'"))
		return
	}

	media, err := app.models.Movies.GetMedia(r.Context(), id, mediaType)
	if err != nil {
		if errors.Is(err, data.ErrRecordNotFound) {
			media, err = app.fetchAndSaveMedia(r.Context(), id, mediaType)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
		} else {
			app.serverErrorResponse(w, r, err)
			return
		}
	}

	if media.Genre == nil {
		media.Genre = []data.Genre{}
	}

	app.writeJSON(w, http.StatusOK, envelope{"media": media}, nil)
}

func (app *application) getMediaSearchHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")
	if query == "" {
		app.badRequestResponse(w, r, errors.New("query parameter is required"))
		return
	}

	url := fmt.Sprintf("%s/3/search/multi?query=%s", app.config.tmdbBaseURL, url.QueryEscape(query))
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+app.config.tmdbToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		app.serverErrorResponse(w, r, errors.New("failed to search TMDB"))
		return
	}
	defer resp.Body.Close()

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

	var searchResults struct {
		Results []TMDBSearchResult `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResults); err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	results := make([]TMDBSearchResult, 0, 20)
	for _, result := range searchResults.Results {
		if result.MediaType == "movie" || result.MediaType == "tv" {
			results = append(results, result)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"media_results": results}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
}

func (app *application) fetchAndSaveMedia(ctx context.Context, tmdbID int32, mediaType string) (*data.Media, error) {
	url := fmt.Sprintf("%s/3/%s/%d", app.config.tmdbBaseURL, mediaType, tmdbID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+app.config.tmdbToken)

	client := &http.Client{Timeout: 5 * time.Second}
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

	err = app.models.Movies.InsertMedia(ctx, newMedia)
	if err != nil {
		return nil, err
	}

	return newMedia, nil
}
