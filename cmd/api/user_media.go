package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
)

const (
	userMediaDefaultLimit int32 = 5
	userMediaMaxLimit     int32 = 20
)

func (app *application) listUserHighestRatedMediaHandler(w http.ResponseWriter, req *http.Request) {
	app.listUserRatedMedia(w, req, true)
}

func (app *application) listUserRecentRatedMediaHandler(w http.ResponseWriter, req *http.Request) {
	app.listUserRatedMedia(w, req, false)
}

func (app *application) listUserRatedMedia(w http.ResponseWriter, req *http.Request, highestRated bool) {
	userID, err := uuid.Parse(req.PathValue("user_id"))
	if err != nil {
		app.badRequestResponse(w, req, fmt.Errorf("invalid user_id"))
		return
	}

	limit := app.readInt(req.URL.Query(), "limit", userMediaDefaultLimit)
	if limit < 1 || limit > userMediaMaxLimit {
		limit = userMediaDefaultLimit
	}

	maxTime := time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	cursorUpdatedAt := maxTime
	cursorMediaID := maxCursorUUID
	cursorRatingValue := int32(11)

	if cursorStr := app.readString(req.URL.Query(), "cursor", ""); cursorStr != "" {
		if highestRated {
			cursorRatingValue, cursorUpdatedAt, cursorMediaID, err = app.decodeCursorHighestRatedMedia(cursorStr)
		} else {
			cursorMediaID, cursorUpdatedAt, err = app.decodeCursor(cursorStr)
		}
		if err != nil {
			app.badRequestResponse(w, req, err)
			return
		}
	}

	var ratedMedia []data.RatedMedia
	if highestRated {
		ratedMedia, err = app.models.Ratings.GetUserHighestRatedMedia(
			req.Context(),
			userID,
			cursorRatingValue,
			cursorUpdatedAt,
			cursorMediaID,
			limit+1,
		)
	} else {
		ratedMedia, err = app.models.Ratings.GetUserRecentRatedMedia(
			req.Context(),
			userID,
			cursorUpdatedAt,
			cursorMediaID,
			limit+1,
		)
	}
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var nextCursor *string
	if len(ratedMedia) > int(limit) {
		ratedMedia = ratedMedia[:limit]
		last := ratedMedia[len(ratedMedia)-1]
		var cursor string
		if highestRated {
			cursor = app.encodeCursorHighestRatedMedia(last.RatingValue, last.RatedAt, last.Media.ID)
		} else {
			cursor = app.encodeCursor(last.Media.ID, last.RatedAt)
		}
		nextCursor = &cursor
	}

	media := make([]data.Media, 0, len(ratedMedia))
	for _, item := range ratedMedia {
		media = append(media, item.Media)
	}

	if err := app.writeJSON(w, http.StatusOK, envelope{
		"media":       media,
		"next_cursor": nextCursor,
	}, nil); err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
