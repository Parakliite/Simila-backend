package main

import (
	"expvar"
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("POST /api/v1/users", app.registerUserHandler)
	mux.HandleFunc("POST /api/v1/login", app.createAuthenticationTokenHandler)
	mux.HandleFunc("PUT /api/v1/users/activated", app.activateUserHandler)
	mux.HandleFunc("GET /api/v1/health", app.healthCheckHandler)

	mux.Handle("GET /api/v1/metrics", expvar.Handler())

	mux.Handle("GET /api/v1/media/{id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getMediaHandler))))

	mux.Handle("GET /api/v1/media/search",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getMediaSearchHandler))))

	mux.Handle("PUT /api/v1/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.upsertRatingHandler))))
	mux.Handle("GET /api/v1/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listRatingsHandler))))
	mux.Handle("GET /api/v1/ratings/{media_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getRatingHandler))))
	mux.Handle("DELETE /api/v1/ratings/{media_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.deleteRatingHandler))))
	mux.Handle("GET /api/v1/media/{media_id}/ratings",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listRatingsForMedia))))

	mux.Handle("PUT /api/v1/reactions",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.upsertReactionHandler))))
	mux.Handle("DELETE /api/v1/reactions/{rating_user_id}/{media_id}",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.deleteReactionHandler))))
	mux.Handle("GET /api/v1/users/{user_id}/reactions",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.listUserReactionsHandler))))
	mux.Handle("GET /api/v1/ratings/{rating_user_id}/{media_id}/reactions/count",
		app.authenticate(http.HandlerFunc(app.requireActivatedUser(app.getReactionCountForRatingHandler))))

	mux.HandleFunc("/", app.notFoundResponse)

	return app.metrics(app.recoverPanic(app.enableCORS(mux)))
}
