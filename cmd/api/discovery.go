package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/parakliite/simila/internal/data"
)

const discoveryLimit int32 = 20

func (app *application) listRandomMediaHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	media, err := app.models.Ratings.GetRandomMedia(req.Context(), user.ID, discoveryLimit+1)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	hasMore := len(media) > int(discoveryLimit)
	if hasMore {
		media = media[:discoveryLimit]
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

		err = app.models.Discovery.SetDiscoveryHistoryExpiry(
			req.Context(),
			user.ID,
			m.ID,
			time.Now().Add(time.Hour*24*7),
		)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"media": parsedMediaList, "has_more": hasMore}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
