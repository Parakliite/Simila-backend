package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/deltron-fr/filmbox/server/internal/data"
)

func (app *application) ListRandomMedia(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	media, err := app.models.Ratings.GetRandomMedia(req.Context(), user.ID, 10)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	var parsedMediaList []data.Media

	for _, m := range media {
		eligible := false
		exp, err := app.models.Discovery.GetDiscoveryHistoryExpiry(req.Context(), user.ID, m.ID)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				eligible = true
			default:
				app.logger.PrintError(err, nil)
				continue
			}
		}

		if !eligible && exp.Before(time.Now()) {
			eligible = true
		}

		if !eligible {
			continue
		}

		parsedMediaList = append(parsedMediaList, m)

		err = app.models.Discovery.SetDiscoveryHistoryExpiry(req.Context(), user.ID, m.ID, time.Now().Add(time.Hour*24*7))
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	}

	app.writeJSON(w, http.StatusOK, envelope{"media": parsedMediaList}, nil)
}
