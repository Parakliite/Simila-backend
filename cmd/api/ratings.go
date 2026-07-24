package main

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) upsertRatingHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	var input struct {
		MediaID     uuid.UUID  `json:"media_id"`
		RatingValue float64    `json:"rating_value"`
		WatchedDate *time.Time `json:"watched_date,omitempty"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	if data.ValidateRatingValue(v, input.RatingValue); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	rating, err := app.models.Ratings.UpsertUserRating(r.Context(), data.Rating{
		MediaID:     input.MediaID,
		UserID:      user.ID,
		RatingValue: input.RatingValue,
		WatchedDate: input.WatchedDate,
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, 200, envelope{"rating": rating}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteRatingHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	err = app.models.Ratings.DeleteUserRating(r.Context(), user.ID, mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, 204, nil, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listRatingsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	qs := r.URL.Query()

	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	cursorStr := app.readString(qs, "cursor", "")

	cursorCreatedAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	cursorMediaID := maxCursorUUID

	if cursorStr != "" {
		var err error
		cursorMediaID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	ratings, err := app.models.Ratings.GetUsersRatings(
		r.Context(),
		cursorCreatedAt,
		limit+1,
		user.ID,
		cursorMediaID,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var nextCursor *string
	if len(ratings) > int(limit) {
		ratings = ratings[:limit]
		last := ratings[limit-1]
		c := app.encodeCursor(last.Media.ID, last.Rating.CreatedAt)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"ratings":     ratings,
		"next_cursor": nextCursor,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getRatingHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	rating, err := app.models.Ratings.GetUsersRating(r.Context(), user.ID, mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"rating": rating}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listRatingsForMedia(w http.ResponseWriter, r *http.Request) {
	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	qs := r.URL.Query()

	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	cursorStr := app.readString(qs, "cursor", "")

	var cursorCreatedAt time.Time
	var cursorUserID uuid.UUID

	if cursorStr != "" {
		var err error
		cursorUserID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, r, err)
			return
		}
	}

	ratings, err := app.models.Ratings.GetAllRatingsForSingleMedia(
		r.Context(),
		cursorCreatedAt,
		mediaID,
		cursorUserID,
		limit+1,
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var nextCursor *string
	if len(ratings) > int(limit) {
		ratings = ratings[:limit]
		last := ratings[limit-1]
		c := app.encodeCursor(last.Rating.UserID, last.Rating.CreatedAt)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"ratings":     ratings,
		"next_cursor": nextCursor,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
