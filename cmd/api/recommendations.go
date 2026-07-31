package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

func (app *application) getUserRecommendations(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	targetUserIDStr := req.PathValue("user_id")
	if targetUserIDStr == "" {
		app.badRequestResponse(w, req, errInvalidUserID)
		return
	}

	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		app.badRequestResponse(w, req, errInvalidUserID)
		return
	}

	if user.ID == targetUserID {
		app.badRequestResponse(w, req, errors.New("invalid request"))
		return
	}

	exists, err := app.models.Matches.CheckMatchExists(req.Context(), user.ID, targetUserID)
	if err != nil {
		app.logger.PrintError(err, map[string]string{"matches": fmt.Sprintf("%v", err)})
		app.serverErrorResponse(w, req, errUnexpected)
		return
	}

	if !exists {
		app.notFoundResponse(w, req)
		return
	}

	limit := app.readInt(req.URL.Query(), "limit", 5)
	if limit < 1 || limit > 10 {
		limit = 5
	}

	recs, err := app.models.Recommendations.GetUserRecommendations(
		req.Context(),
		user.ID,
		targetUserID,
		7,
		limit,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"recommendations": recs}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
