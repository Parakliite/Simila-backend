package main

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/parakliite/simila/internal/data"
	"github.com/parakliite/simila/internal/validator"
)

func (app *application) getUserProfileHandler(w http.ResponseWriter, req *http.Request) {
	authenticatedUser := app.contextGetUser(req)

	user, err := app.models.Users.GetByID(req.Context(), authenticatedUser.ID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) updateUserProfileHandler(w http.ResponseWriter, req *http.Request) {
	user := app.contextGetUser(req)

	var input struct {
		Name              *string `json:"name"`
		About             *string `json:"about"`
		ProfilePictureURL *string `json:"profile_picture_url"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()
	if input.Name == nil && input.About == nil && input.ProfilePictureURL == nil {
		v.AddError("body", "must provide at least one field to update")
	}

	if input.Name != nil {
		v.Check(*input.Name != "", "name", "must be provided")
		v.Check(len(*input.Name) <= 500, "name", "must not be more than 500 bytes long")
	}

	if !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.About != nil {
		user.About = *input.About
	}
	if input.ProfilePictureURL != nil {
		user.ProfilePictureURL = *input.ProfilePictureURL
	}

	err = app.models.Users.UpdateUser(req.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, req, v.Errors)
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) registerUserHandler(w http.ResponseWriter, req *http.Request) {
	// an anonymous struct to hold the request data from the expected body
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	err = app.models.Users.Insert(req.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, req, v.Errors)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	token, err := app.models.Tokens.New(
		req.Context(),
		user.ID,
		uuid.NullUUID{Valid: false},
		24*time.Hour,
		data.ScopeActivation,
	)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	app.background(func() {
		data := map[string]interface{}{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		err = app.mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			app.logger.PrintError(err, nil)
		}
	})

	err = app.writeJSON(w, 202, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, req *http.Request) {
	var input struct {
		TokenPlaintext string `json:"token"`
	}

	err := app.readJSON(w, req, &input)
	if err != nil {
		app.badRequestResponse(w, req, err)
		return
	}

	v := validator.New()

	if data.ValidateTokenPlaintext(v, input.TokenPlaintext); !v.Valid() {
		app.failedValidationResponse(w, req, v.Errors)
		return
	}

	user, err := app.models.Users.GetUserForToken(req.Context(), data.ScopeActivation, input.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation tokens")
			app.failedValidationResponse(w, req, v.Errors)
		default:
			app.serverErrorResponse(w, req, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.UpdateUser(req.Context(), user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, req)
		default:
			app.serverErrorResponse(w, req, err)
		}
	}

	err = app.models.Tokens.DeleteAllForUser(req.Context(), data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, req, err)
		return
	}

	// Send the updated user details to the client in a JSON response.
	err = app.writeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, req, err)
	}
}
