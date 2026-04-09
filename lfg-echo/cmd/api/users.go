package main

import (
	"errors"
	"time"

	"github.com/hbasaum/lfg-echo/internal/data"
	"github.com/hbasaum/lfg-echo/internal/validator"
	"github.com/labstack/echo/v4"
)

type registerUserRequest struct {
	Name     string `json:"name" validate:"required,max=500"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=72"`
}

type activateUserRequest struct {
	TokenPlainText string `json:"token" validate:"required,len=26"`
}

func (app *application) registerUserHanlder(c echo.Context) error {
	var input registerUserRequest
	if err := app.bindAndValidate(c, &input); err != nil {
		return err
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err := user.Password.Set(input.Password)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		return app.failedValidationResponse(c, v.Errors)
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			return app.failedValidationResponse(c, map[string]string{
				"email": "a user with this email address already exists",
			})
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	// Keep background email send non-blocking so registration latency stays low.
	app.background(func() {
		payload := map[string]any{
			"activationToken": token.Plaintext,
			"userID":          user.ID,
		}

		if sendErr := app.mailer.Send(user.Email, "user_welcome.tmpl", payload); sendErr != nil {
			app.logger.Error(sendErr.Error())
		}
	})

	return c.JSON(202, envelope{"user": user})
}

func (app *application) activateUserHandler(c echo.Context) error {
	var input activateUserRequest
	if err := app.bindAndValidate(c, &input); err != nil {
		return err
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenPlainText)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return app.failedValidationResponse(c, map[string]string{
				"token": "invalid token or expired activation token",
			})
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	user.Activated = true
	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			return app.editConflictResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	err = app.models.Tokens.DeleteAllForUser(data.ScopeActivation, user.ID)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	return c.JSON(200, envelope{"user": user})
}
