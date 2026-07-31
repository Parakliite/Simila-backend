package main

import (
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/validator"
)

const (
	accessTokenExpiryDuration  = 1 * time.Hour
	refreshTokenExpiryDuration = 24 * 31 * time.Hour
)

func (app *application) loginHandler(w http.ResponseWriter, req *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()

	data.ValidateEmail(v, input.Email)
	data.ValidatePasswordPlaintext(v, input.Password)

	if !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	user, err := app.models.Users.GetByEmail(
		req.Context(),
		input.Email,
	)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialsResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	if !match {
		app.invalidCredentialsResponse(w, req)
		return
	}

	// creates a new session id for each login
	sessionID := uuid.New()

	accessToken, err := app.models.Tokens.New(
		req.Context(),
		user.ID,
		uuid.NullUUID{Valid: false},
		accessTokenExpiryDuration,
		data.ScopeAuthentication,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.models.Tokens.RevokeAllPreviousTokens(req.Context(), data.ScopeRefresh, user.ID)
	if err != nil {
		app.serverErrorResponse(w, req, err) // does it return an error if no rows exist?
		return
	}

	refreshToken, err := app.models.Tokens.New(
		req.Context(),
		user.ID,
		uuid.NullUUID{UUID: sessionID, Valid: true},
		refreshTokenExpiryDuration,
		data.ScopeRefresh,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(
		w,
		http.StatusCreated,
		envelope{"access_token": accessToken, "refresh_token": refreshToken},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) refreshHandler(w http.ResponseWriter, req *http.Request) {
	refreshHeader := req.Header.Get("Authorization")

	if refreshHeader == "" {
		app.invalidRefreshTokenResponse(w, req)
		return
	}

	headerParts := strings.Split(refreshHeader, " ")
	if len(headerParts) != 2 || headerParts[0] != "Bearer" {
		app.invalidRefreshTokenResponse(w, req)
		return
	}

	refreshToken := headerParts[1]
	v := validator.New()

	if data.ValidateTokenPlaintext(v, refreshToken); !v.Valid() {
		app.invalidRefreshTokenResponse(w, req)
		return
	}

	hash := sha256.Sum256([]byte(refreshToken))
	refreshTokenHash := hash[:]

	// Get the user from the refresh token
	userID, sessionID, err := app.models.Tokens.GetUserIDFromToken(
		req.Context(),
		data.ScopeRefresh,
		refreshTokenHash)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidRefreshTokenResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	// create a new access token
	accessToken, err := app.models.Tokens.New(
		req.Context(),
		userID,
		uuid.NullUUID{Valid: false},
		accessTokenExpiryDuration,
		data.ScopeAuthentication,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	// revoke previous refresh token
	err = app.models.Tokens.RevokePreviousToken(req.Context(), data.ScopeRefresh, refreshTokenHash)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidRefreshTokenResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	// create new refresh token
	newRefreshToken, err := app.models.Tokens.New(
		req.Context(),
		userID,
		uuid.NullUUID{UUID: sessionID, Valid: true},
		refreshTokenExpiryDuration,
		data.ScopeRefresh,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	err = app.writeJSON(
		w,
		http.StatusCreated,
		envelope{"access_token": accessToken, "refresh_token": newRefreshToken},
		nil,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
