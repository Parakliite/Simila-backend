package main

import (
	"net/http"
)

func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/media/{id}", app.getMediaHandler)
	mux.HandleFunc("GET /api/v1/media/search", app.getMediaSearchHandler)

	// USERS
	mux.HandleFunc("POST /api/v1/users", app.registerUserHandler)
	mux.HandleFunc("PUT /api/v1/users/activated", app.activateUserHandler)
	mux.HandleFunc("POST /api/v1/tokens/authentication", app.activateUserHandler)

	mux.HandleFunc("GET /api/v1/health", app.healthCheckHandler)

	mux.HandleFunc("/", app.notFoundResponse)

	return app.recoverPanic(app.authenticate(mux))
}
