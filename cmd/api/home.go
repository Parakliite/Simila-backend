package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/parakliite/simila/internal/data"
)

type trendingResult struct {
	TmdbID      int32  `json:"tmdb_id"`
	Title       string `json:"title"`
	PosterPath  string `json:"poster_path"`
	MediaType   string `json:"media_type"`
	Overview    string `json:"overview"`
	ReleaseDate string `json:"release_date"`
}

func (app *application) getTopRatedByMatchesHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 10)
	if limit < 1 || limit > 20 {
		limit = 10
	}

	results, err := app.models.Home.GetTopRatedByMatches(req.Context(), user.ID, limit)
	if err != nil {
		if !errors.Is(err, data.ErrRecordNotFound) {
			app.serverErrorResponse(w, req, err)
			return
		}
	}

	if results == nil {
		results = []data.MatchRatedMedia{}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"top_rated_by_matches": results}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) getTrendingHandler(w http.ResponseWriter, req *http.Request) {
	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 10)
	if limit < 1 || limit > 20 {
		limit = 10
	}

	tmdbReq, err := app.newTMDBRequest(req.Context(), []string{"3", "trending", "all", "week"}, url.Values{})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(tmdbReq)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		app.serverErrorResponse(w, req, errors.New("failed to fetch trending from TMDB"))
		return
	}

	var tmdbResp struct {
		Results []struct {
			ID           int32  `json:"id"`
			Title        string `json:"title"`
			Name         string `json:"name"`
			PosterPath   string `json:"poster_path"`
			MediaType    string `json:"media_type"`
			Overview     string `json:"overview"`
			ReleaseDate  string `json:"release_date"`
			FirstAirDate string `json:"first_air_date"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tmdbResp); err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var results []trendingResult
	for _, r := range tmdbResp.Results {
		if r.MediaType != "movie" && r.MediaType != "tv" {
			continue
		}
		title := r.Title
		if title == "" {
			title = r.Name
		}
		releaseDate := r.ReleaseDate
		if releaseDate == "" {
			releaseDate = r.FirstAirDate
		}
		results = append(results, trendingResult{
			TmdbID:      r.ID,
			Title:       title,
			PosterPath:  r.PosterPath,
			MediaType:   r.MediaType,
			Overview:    r.Overview,
			ReleaseDate: releaseDate,
		})
		if len(results) >= int(limit) {
			break
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"trending": results}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
