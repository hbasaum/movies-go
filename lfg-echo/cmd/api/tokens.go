package main

import (
	"errors"
	"time"

	"github.com/hbasaum/lfg-echo/internal/data"
	"github.com/labstack/echo/v4"
)

type createAuthenticationTokenRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

func (app *application) createAuthenticationTokenHandler(c echo.Context) error {
	var input createAuthenticationTokenRequest
	if err := app.bindAndValidate(c, &input); err != nil {
		return err
	}

	user, err := app.models.Users.GetByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return app.invalidCredentialsResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	match, err := user.Password.Matches(input.Password)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}
	if !match {
		return app.invalidCredentialsResponse(c)
	}

	token, err := app.models.Tokens.New(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	return c.JSON(201, envelope{"authentication_token": token})
}
