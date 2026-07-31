package main

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const matchRecalculationInterval = 1 * time.Hour

func (app *application) recalculateMatchesJob(ctx context.Context) {
	ticker := time.NewTicker(matchRecalculationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			app.logger.PrintInfo("Worker stopped by context cancellation", nil)
			return
		case <-ticker.C:
			err := app.recalculateMatches(ctx)
			if err != nil {
				app.logger.PrintError(err, map[string]string{"worker": "stopped on error"})
				return
			}
		}
	}
}

func (app *application) recalculateMatches(ctx context.Context) error {
	indexmap, err := app.models.Matches.MapMediaToIndex(ctx)
	if err != nil {
		return err
	}
	if len(indexmap) == 0 {
		return nil
	}

	allUsers, err := app.models.Users.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	for _, userID := range allUsers {
		ratings, err := app.models.Matches.GetAllUserRatingsForMatches(ctx, userID)
		if err != nil {
			return err
		}

		err = app.models.Matches.CalculateSimilarities(
			ctx,
			userID,
			ratings,
			indexmap,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (app *application) listMatchesHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	qs := req.URL.Query()
	limit := app.readInt(qs, "limit", 5)
	if limit < 1 || limit > 20 {
		limit = 5
	}

	cursorStr := app.readString(qs, "cursor", "")
	var cursorScore sql.NullFloat64
	var cursorSharedMediaCount sql.NullInt32
	var cursorLastRecalculatedAt sql.NullTime
	var cursorOtherUserID uuid.NullUUID

	if cursorStr != "" {
		score, sharedMediaCount, lastRecalculatedAt, otherUserID, err := app.decodeCursorMatches(cursorStr)
		if err != nil {
			app.badRequestResponse(w, req, err)
			return
		}
		cursorScore = sql.NullFloat64{
			Valid:   true,
			Float64: score,
		}
		cursorSharedMediaCount = sql.NullInt32{
			Valid: true,
			Int32: sharedMediaCount,
		}
		cursorLastRecalculatedAt = sql.NullTime{
			Valid: true,
			Time:  lastRecalculatedAt,
		}
		cursorOtherUserID = uuid.NullUUID{
			Valid: true,
			UUID:  otherUserID,
		}
	}

	matches, err := app.models.Matches.GetUserMatches(
		req.Context(),
		user.ID,
		limit+1,
		cursorOtherUserID,
		cursorScore,
		cursorSharedMediaCount,
		cursorLastRecalculatedAt,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var nextCursor *string
	if len(matches) > int(limit) {
		matches = matches[:limit]
		last := matches[len(matches)-1]
		c := app.encodeCursorMatches(
			last.Score,
			last.SharedMedia,
			last.LastRecalculatedAt,
			last.User.ID,
		)
		nextCursor = &c
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"matches": matches, "next_cursor": nextCursor}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
