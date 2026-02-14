package main

import (
	"net/http"
)

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v1/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello, filmbox!"))
	})
	mux.HandleFunc("GET /v1/movies", app.getMoviesHandler)
	mux.HandleFunc("/", app.notFoundResponse)

	return mux

}