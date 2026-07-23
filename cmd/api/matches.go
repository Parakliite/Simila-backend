package main

import "net/http"

func (app *application) listMatchesHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	indexmap, err := app.models.Matches.MapMediaToIndex()
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	ratings, err := app.models.Matches.GetAllUserRatingsForMatches(req.Context(), user.ID)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	similarities, err := app.models.Matches.GetSimilarities(
		req.Context(),
		user.ID,
		ratings,
		indexmap)
	if err != nil {
		// TODO: do something
		return
	}

	app.writeJSON(w, http.StatusOK, envelope{"matches": similarities}, nil)
}
