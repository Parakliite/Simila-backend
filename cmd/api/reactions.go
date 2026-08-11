package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/database"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) upsertReactionHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	var input struct {
		RatingUserID uuid.UUID `json:"rating_user_id"`
		TmdbID       int32     `json:"tmdb_id"`
		MediaType    string    `json:"media_type"`
		Reaction     string    `json:"reaction"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	data.ValidateReactionType(v, input.Reaction)
	if input.RatingUserID == uuid.Nil {
		v.AddError("rating_user_id", "must be provided")
	}
	if input.TmdbID <= 0 {
		v.AddError("tmdb_id", "must be a positive integer")
	}
	if input.MediaType != "movie" && input.MediaType != "tv" {
		v.AddError("media_type", "must be 'movie' or 'tv'")
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

	reaction, err := app.models.Reactions.UpsertReaction(req.Context(), data.Reaction{
		ReactorUserID: user.ID,
		RatingUserID:  input.RatingUserID,
		MediaID:       mediaID,
		Reaction:      database.ReactionType(input.Reaction),
	})
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reaction": reaction}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) deleteReactionHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	ratingUserID, err := uuid.Parse(req.PathValue("rating_user_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid rating_user_id"))
		return
	}

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

	err = app.models.Reactions.DeleteReaction(req.Context(), user.ID, ratingUserID, mediaID)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
			app.notFoundResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *application) listUserReactionsHandler(w http.ResponseWriter, req *http.Request) {
	ratingUserID, err := uuid.Parse(req.PathValue("user_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid user_id"))
		return
	}

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 20)
	if limit < 1 || limit > 70 {
		limit = 20
	}

	cursorStr := app.readString(qs, "cursor", "")
	cursorCreatedAt := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	cursorReactorUserID := maxCursorUUID

	if cursorStr != "" {
		cursorReactorUserID, cursorCreatedAt, err = app.decodeCursor(cursorStr)
		if err != nil {
			app.badRequestResponse(w, req, err)
			return
		}
	}

	reactions, err := app.models.Reactions.GetUserReactionsForTargetUser(
		req.Context(),
		ratingUserID,
		cursorCreatedAt,
		cursorReactorUserID,
		limit+1,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var nextCursor *string
	if len(reactions) > int(limit) {
		reactions = reactions[:limit]
		last := reactions[len(reactions)-1]
		c := app.encodeCursor(last.Reactor.ID, last.CreatedAt)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"reactions":   reactions,
		"next_cursor": nextCursor,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) getReactionCountForRatingHandler(w http.ResponseWriter, req *http.Request) {
	ratingUserID, err := uuid.Parse(req.PathValue("rating_user_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid rating_user_id"))
		return
	}

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

	count, err := app.models.Reactions.GetReactionCountForRating(req.Context(), ratingUserID, mediaID)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reaction_count": count}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
