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

	mux.HandleFunc("/", app.notFoundResponse)

	return app.metrics(app.recoverPanic(app.enableCORS(mux)))
}
