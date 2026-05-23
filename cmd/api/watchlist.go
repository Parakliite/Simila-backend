package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

func (app *application) insertToWatchlistHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		MediaID       uuid.UUID  `json:"media_id"`
		Source        string     `json:"source"`
		SourceMatchID *uuid.UUID `json:"source_match_id,omitempty"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
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
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	watchlist, err := app.models.Watchlist.InsertMediaToWatchlist(r.Context(), data.Watchlist{
		UserID:        user.ID,
		MediaID:       input.MediaID,
		Source:        input.Source,
		SourceMatchID: input.SourceMatchID,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"watchlist": watchlist}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateWatchlistStatusHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	var input struct {
		Status string `json:"status"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateWatchlistStatus(v, input.Status)
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	watchlist, err := app.models.Watchlist.UpdateWatchlistItemStatus(r.Context(), user.ID, mediaID, input.Status)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"watchlist": watchlist}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteWatchlistItemHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	err = app.models.Watchlist.DeleteMediaFromWatchlist(r.Context(), user.ID, mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusNoContent, nil, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listWatchlistItemsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	qs := r.URL.Query()
	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	cursorStr := app.readString(qs, "cursor", "")
	var cursorCreatedAt time.Time
	var cursorMediaID uuid.UUID

	if cursorStr != "" {
		var err error
		cursorMediaID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	items, err := app.models.Watchlist.GetAllItemsInWatchlist(
		r.Context(),
		cursorCreatedAt,
		int32(limit+1),
		user.ID,
		cursorMediaID,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var nextCursor *string
	if len(items) > limit {
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
		app.serverErrorResponse(w, r, err)
	}
}
