package main

import (
	"net/http"
)

func (app *application) routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("Hello, filmbox!"))
	})
	mux.HandleFunc("GET /media/{id}", app.getMediaHandler)
	mux.HandleFunc("GET /media/search", app.getMediaSearchHandler)
	mux.HandleFunc("GET /health", app.healthCheckHandler)
	
	mux.HandleFunc("/", app.notFoundResponse)

	return mux

}
