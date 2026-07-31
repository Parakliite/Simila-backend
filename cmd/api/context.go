package main

import (
	"context"
	"net/http"

	"github.com/parakliite/simila/internal/data"
)

type contextKey string

const userContextKey = contextKey("user")

func (app *application) contextSetUser(req *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(req.Context(), userContextKey, user)
	return req.WithContext(ctx)
}

func (app *application) contextGetUser(req *http.Request) *data.User {
	user, ok := req.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
