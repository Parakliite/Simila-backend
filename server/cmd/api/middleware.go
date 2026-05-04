package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/deltron-fr/filmbox/server/internal/data"
	"github.com/deltron-fr/filmbox/server/internal/validator"
)

func (app *application) recoverPanic(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Create a deferred function (which will always be run in the event of a panic
        // as Go unwinds the stack).
        defer func() {
            // Use the builtin recover function to check if there has been a panic or
            // not.
            if err := recover(); err != nil {
                // If there was a panic, set a "Connection: close" header on the 
                // response. This acts as a trigger to make Go's HTTP server 
                // automatically close the current connection after a response has been 
                // sent.
                w.Header().Set("Connection", "close")
                // The value returned by recover() has the type interface{}, so we use
                // fmt.Errorf() to normalize it into an error and call our 
                // serverErrorResponse() helper. In turn, this will log the error using
                // our custom Logger type at the ERROR level and send the client a 500
                // Internal Server Error response.
                app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
            }
        }()
        next.ServeHTTP(w, r)
    })
}

func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		// Add the "Vary: Authorization" header to the response. This indicates to any
		// caches that the response may vary based on the value of the Authorization
		// header in the request.
		// w.Header().Add("Vary", "Authorization")

		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			app.invalidAuthenticationTokenResponse(w, r)
			return 
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return 
		}

		token := headerParts[1]
		v := validator.New()

		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return 
		}

		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return 
		}

		r = app.contextSetUser(r, user)

		next.ServeHTTP(w, r)
	})
}
