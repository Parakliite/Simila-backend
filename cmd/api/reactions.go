package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/database"
	"github.com/deltron-fr/filmbox/server/internal/validator"
	"github.com/google/uuid"
)

var maxCursorUUID = uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")

func (app *application) upsertReactionHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	var input struct {
		RatingUserID uuid.UUID `json:"rating_user_id"`
		MediaID      uuid.UUID `json:"media_id"`
		Reaction     string    `json:"reaction"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	v := validator.New()
	data.ValidateReactionType(v, input.Reaction)
	if input.RatingUserID == uuid.Nil {
		v.AddError("rating_user_id", "must be provided")
	}
	if input.MediaID == uuid.Nil {
		v.AddError("media_id", "must be provided")
	}
	if !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	reaction, err := app.models.Reactions.UpsertReaction(r.Context(), data.Reaction{
		ReactorUserID: user.ID,
		RatingUserID:  input.RatingUserID,
		MediaID:       input.MediaID,
		Reaction:      database.ReactionType(input.Reaction),
	})
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reaction": reaction}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteReactionHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)

	ratingUserID, err := uuid.Parse(r.PathValue("rating_user_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid rating_user_id"))
		return
	}

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	err = app.models.Reactions.DeleteReaction(r.Context(), user.ID, ratingUserID, mediaID)
	if err != nil {
		switch err {
		case data.ErrRecordNotFound:
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

func (app *application) listUserReactionsHandler(w http.ResponseWriter, r *http.Request) {
	ratingUserID, err := uuid.Parse(r.PathValue("user_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid user_id"))
		return
	}

	qs := r.URL.Query()
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
			app.badRequestResponse(w, r, err)
			return
		}
	}

	reactions, err := app.models.Reactions.GetUserReactionsForTargetUser(
		r.Context(),
		ratingUserID,
		cursorCreatedAt,
		cursorReactorUserID,
		int32(limit+1),
	)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	var nextCursor *string
	if len(reactions) > limit {
		reactions = reactions[:limit]
		last := reactions[len(reactions)-1]
		c := app.encodeCursor(last.ReactorUserID, last.CreatedAt)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{
		"reactions":   reactions,
		"next_cursor": nextCursor,
	}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) getReactionCountForRatingHandler(w http.ResponseWriter, r *http.Request) {
	ratingUserID, err := uuid.Parse(r.PathValue("rating_user_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid rating_user_id"))
		return
	}

	mediaID, err := uuid.Parse(r.PathValue("media_id"))
	if err != nil {
		app.badRequestResponse(w, r, fmt.Errorf("invalid media_id"))
		return
	}

	count, err := app.models.Reactions.GetReactionCountForRating(r.Context(), ratingUserID, mediaID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"reaction_count": count}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
