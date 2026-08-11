package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) insertToWatchlistHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	var input struct {
		TmdbID        int32      `json:"tmdb_id"`
		MediaType     string     `json:"media_type"`
		Source        string     `json:"source"`
		SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	if input.TmdbID <= 0 {
		v.AddError("tmdb_id", "must be a positive integer")
	}
	if input.MediaType != "movie" && input.MediaType != "tv" {
		v.AddError("media_type", "must be 'movie' or 'tv'")
	}
	data.ValidateWatchlistSource(v, input.Source)
	if input.Source == string(database.SourceTypeFromMatch) && input.SourceMatchID == nil {
		v.AddError("source_match_id", "must be provided when source is from_match")
	}
	if input.Source == string(database.SourceTypeSelf) && input.SourceMatchID != nil {
		v.AddError("source_match_id", "must not be provided when source is self")
	}
	if !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(req.Context(), input.TmdbID, input.MediaType)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	watchlist, err := app.models.Watchlist.InsertMediaToWatchlist(req.Context(), data.Watchlist{
		UserID:        user.ID,
		MediaID:       mediaID,
		Source:        input.Source,
		SourceMatchID: input.SourceMatchID,
	})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"watchlist": watchlist}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) updateWatchlistStatusHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	tmdbID, err := strconv.ParseInt(req.PathValue("tmdb_id"), 10, 32)
	if err != nil || tmdbID < 1 {
		app.badRequestResponse(w, req, fmt.Errorf("invalid tmdb_id"))
		return
	}

	mediaType := req.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, req, fmt.Errorf("type parameter must be 'movie' or 'tv'"))
		return
	}

	mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(req.Context(), int32(tmdbID), mediaType)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var input struct {
		Status string `json:"status"`
	}

	err = app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	data.ValidateWatchlistStatus(v, input.Status)
	if !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	watchlist, err := app.models.Watchlist.UpdateWatchlistItemStatus(
		req.Context(),
		user.ID,
		mediaID,
		input.Status,
	)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"watchlist": watchlist}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) deleteWatchlistItemHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	tmdbID, err := strconv.ParseInt(req.PathValue("tmdb_id"), 10, 32)
	if err != nil || tmdbID < 1 {
		app.badRequestResponse(w, req, fmt.Errorf("invalid tmdb_id"))
		return
	}

	mediaType := req.URL.Query().Get("type")
	if mediaType != "movie" && mediaType != "tv" {
		app.badRequestResponse(w, req, fmt.Errorf("type parameter must be 'movie' or 'tv'"))
		return
	}

	mediaID, err := app.models.Movies.GetOrCreateMediaByTmdbID(req.Context(), int32(tmdbID), mediaType)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.models.Watchlist.DeleteMediaFromWatchlist(req.Context(), user.ID, mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) listWatchlistItemsHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	mediaStatus := app.readString(qs, "status", "")

	cursorStr := app.readString(qs, "cursor", "")
	var cursorCreatedAt time.Time
	var cursorMediaID uuid.UUID

	if cursorStr != "" {
		var err error
		cursorMediaID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, req, err)
			return
		}
	}

	items, err := app.models.Watchlist.GetAllItemsInWatchlist(
		req.Context(),
		cursorCreatedAt,
		limit+1,
		mediaStatus,
		user.ID,
		cursorMediaID,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var nextCursor *string
	if len(items) > int(limit) {
		items = items[:limit]
		last := items[len(items)-1]
		c := app.encodeCursor(last.Media.ID, last.Watchlist.CreatedAt)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"watchlist_items": items,
		"next_cursor":     nextCursor,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
