package main

import (
	"github.com/hbasaum/lfg-echo/internal/data"
	"github.com/labstack/echo/v4"
)

type contextKey string

const userContextKey = contextKey("user")

func (app *application) contextSetUser(c echo.Context, user *data.User) {
	c.Set(string(userContextKey), user)
}

func (app *application) contextGetUser(c echo.Context) *data.User {
	// Authentication middleware guarantees the user context key is always set.
	user, ok := c.Get(string(userContextKey)).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}

	return user
}
