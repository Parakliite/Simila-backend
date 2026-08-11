package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) upsertRatingHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)
	var input struct {
		TmdbID      int32      `json:"tmdb_id"`
		MediaType   string     `json:"media_type"`
		RatingValue float64    `json:"rating_value"`
		WatchedDate *time.Time `json:"watched_date,omitempty"`
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
	if data.ValidateRatingValue(v, input.RatingValue); !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
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

	rating, err := app.models.Ratings.UpsertUserRating(req.Context(), data.Rating{
		MediaID:     mediaID,
		UserID:      user.ID,
		RatingValue: input.RatingValue,
		WatchedDate: input.WatchedDate,
	})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, 200, envelope{"rating": rating}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) deleteRatingHandler(w http.ResponseWriter, req *http.Request) {
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

	err = app.models.Ratings.DeleteUserRating(req.Context(), user.ID, mediaID)
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

func (app *application) listRatingsHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	qs := req.URL.Query()

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
			app.badRequestResponse(w, req, err)
			return
		}
	}

	ratings, err := app.models.Ratings.GetUsersRatings(
		req.Context(),
		cursorCreatedAt,
		limit+1,
		user.ID,
		cursorMediaID,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
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
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) getRatingHandler(w http.ResponseWriter, req *http.Request) {
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

	rating, err := app.models.Ratings.GetUsersRating(req.Context(), user.ID, mediaID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"rating": rating}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) listRatingsForMedia(w http.ResponseWriter, req *http.Request) {
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

	qs := req.URL.Query()

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
			app.badRequestResponse(w, req, err)
			return
		}
	}

	ratings, err := app.models.Ratings.GetAllRatingsForSingleMedia(
		req.Context(),
		cursorCreatedAt,
		mediaID,
		cursorUserID,
		limit+1,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
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
		app.serverErrorResponse(w, req, err)
	}
}
