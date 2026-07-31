package main

import (
	"errors"
	"net/http"
)

var (
	errInvalidUserID = errors.New("an invalid user id was passed")
	errUnexpected    = errors.New("an unexpected error occurred")
)

func (app *application) logError(req *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": req.Method,
		"request_url":    req.URL.String(),
	})
}

func (app *application) errorResponse(
	w http.ResponseWriter,
	req *http.Request,
	status int,
	message interface{},
) {
	// Reuse the normal JSON writer so error payloads follow the same response path.
	err := app.writeJSON(w, status, envelope{"error": message}, nil)
	if err != nil {
		w.WriteHeader(500)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, req *http.Request, err error) {
	app.logError(req, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, req, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, req *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, req, http.StatusNotFound, message)
}

/*
func (app *application) methodNotAllowedResponnse(w http.ResponseWriter, req *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", req.Method)
	app.errorResponse(w, req, http.StatusMethodNotAllowed, message)
}
*/

func (app *application) badRequestResponse(w http.ResponseWriter, req *http.Request, err error) {
	app.errorResponse(w, req, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(
	w http.ResponseWriter,
	req *http.Request,
	errors map[string]string,
) {
	app.errorResponse(w, req, http.StatusUnprocessableEntity, errors)
}

func (app *application) editConflictResponse(w http.ResponseWriter, req *http.Request) {
	app.errorResponse(w, req, http.StatusConflict, "error editing...")
}

func (app *application) invalidCredentialsResponse(w http.ResponseWriter, req *http.Request) {
	message := "invalid authentication credentials"
	app.errorResponse(w, req, http.StatusUnauthorized, message)
}

func (app *application) invalidAuthenticationTokenResponse(w http.ResponseWriter, req *http.Request) {
	// Tell clients to authenticate with a bearer token, even when the token is
	// missing rather than merely invalid.
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	app.errorResponse(w, req, http.StatusUnauthorized, message)
}

func (app *application) invalidRefreshTokenResponse(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing refresh token"
	app.errorResponse(w, req, http.StatusBadRequest, message)
}

/*
func (app *application) authenticationRequiredResponse(w http.ResponseWriter, req *http.Request) {
	message := "you must be authenticated to access this resource"
	app.errorResponse(w, req, http.StatusUnauthorized, message)
}
*/

func (app *application) inactiveAccountResponse(w http.ResponseWriter, req *http.Request) {
	message := "your user account must be activated to access this resource"
	app.errorResponse(w, req, http.StatusForbidden, message)
}

func (app *application) unauthorizedRequestResponse(w http.ResponseWriter, req *http.Request) {
	message := "you can't access this"
	app.errorResponse(w, req, http.StatusForbidden, message)
}
