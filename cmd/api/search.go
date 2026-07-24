package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/parakliite/simila/internal/data"
	"golang.org/x/sync/errgroup"
)

func (app *application) searchHandler(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query().Get("query")
	if query == "" {
		app.badRequestResponse(w, req, errors.New("query parameter is required"))
		return
	}

	var limit int32
	qs := req.URL.Query()
	limit = app.readInt(qs, "limit", 10)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	var users []data.User
	var media_results []TMDBSearchResult

	g, ctx := errgroup.WithContext(req.Context())

	g.Go(func() error {
		var err error
		media_results, err = app.mediaSearch(ctx, query)
		return err
	})

	g.Go(func() error {
		var err error
		users, err = app.userSearch(ctx, query, limit)
		return err
	})

	if err := g.Wait(); err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err := app.writeJSON(w, http.StatusOK, envelope{"media_results": media_results, "users_results": users}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) userSearch(ctx context.Context, query string, limit int32) ([]data.User, error) {
	users, err := app.models.Users.SearchUsers(ctx, query, limit)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (app *application) mediaSearch(ctx context.Context, query string) ([]TMDBSearchResult, error) {
	req, err := app.newTMDBRequest(ctx, []string{"3", "search", "multi"}, url.Values{
		"query": []string{query},
	})
	if err != nil {
		return nil, err
	}

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
