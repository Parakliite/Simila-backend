package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) insertToWatchlistHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	var input struct {
		MediaID       uuid.UUID  `json:"media_id"`
		Source        string     `json:"source"`
		SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	if input.MediaID == uuid.Nil {
		v.AddError("media_id", "must be provided")
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

	watchlist, err := app.models.Watchlist.InsertMediaToWatchlist(req.Context(), data.Watchlist{
		UserID:        user.ID,
		MediaID:       input.MediaID,
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

	mediaID, err := uuid.Parse(req.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid media_id"))
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

	mediaID, err := uuid.Parse(req.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid media_id"))
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
