package main

import (
	"errors"
	"fmt"
	"net/http"

	govalidator "github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

func (app *application) logError(c echo.Context, err error) {
	method := c.Request().Method
	uri := c.Request().URL.RequestURI()
	app.logger.Error(err.Error(), "method", method, "uri", uri)
}

func (app *application) errorResponse(c echo.Context, status int, message any) error {
	return c.JSON(status, envelope{"error": message})
}

func (app *application) serverErrorResponse(c echo.Context, err error) error {
	app.logError(c, err)
	message := "the server encountered a problem and could not process your request"
	return app.errorResponse(c, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(c echo.Context) error {
	message := "the requested resource could not be found"
	return app.errorResponse(c, http.StatusNotFound, message)
}

func (app *application) methodNotAllowedResponse(c echo.Context) error {
	message := fmt.Sprintf("the %s method is not supported for this resource", c.Request().Method)
	return app.errorResponse(c, http.StatusMethodNotAllowed, message)
}

func (app *application) badRequestResponse(c echo.Context, err error) error {
	return app.errorResponse(c, http.StatusBadRequest, err.Error())
}

func (app *application) failedValidationResponse(c echo.Context, validationErrors map[string]string) error {
	return app.errorResponse(c, http.StatusUnprocessableEntity, validationErrors)
}

func (app *application) bindAndValidate(c echo.Context, dst any) error {
	if err := c.Bind(dst); err != nil {
		return app.badRequestResponse(c, err)
	}

	if err := c.Validate(dst); err != nil {
		var validationErrors govalidator.ValidationErrors
		if errors.As(err, &validationErrors) {
			formatted := make(map[string]string, len(validationErrors))
			for _, fieldErr := range validationErrors {
				fieldName := normalizeFieldName(fieldErr.Field())
				formatted[fieldName] = validationMessage(fieldErr)
			}

			return app.failedValidationResponse(c, formatted)
		}

		return app.badRequestResponse(c, err)
	}

	return nil
}

func (app *application) editConflictResponse(c echo.Context) error {
	message := "unable to update the record due to an edit conflict, please try again"
	return app.errorResponse(c, http.StatusConflict, message)
}

func (app *application) rateLimitExceededResponse(c echo.Context) error {
	message := "rate limit exceeded"
	return app.errorResponse(c, http.StatusTooManyRequests, message)
}

func (app *application) invalidCredentialsResponse(c echo.Context) error {
	message := "invalid authentication credentials"
	return app.errorResponse(c, http.StatusUnauthorized, message)
}

func (app *application) invalidAuthenticationTokenResponse(c echo.Context) error {
	c.Response().Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	return app.errorResponse(c, http.StatusUnauthorized, message)
}

func (app *application) authenticationRequiredResponse(c echo.Context) error {
	message := "you must be authenticated to access this resource"
	return app.errorResponse(c, http.StatusUnauthorized, message)
}

func (app *application) inactiveAccountResponse(c echo.Context) error {
	message := "your user account must be activated to access this resource"
	return app.errorResponse(c, http.StatusForbidden, message)
}

func (app *application) notPermittedResopnse(c echo.Context) error {
	message := "your user account doesn't have the necessary permissions to access this resource"
	return app.errorResponse(c, http.StatusForbidden, message)
}

func (app *application) httpErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	var httpErr *echo.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Code {
		case http.StatusNotFound:
			_ = app.notFoundResponse(c)
			return
		case http.StatusMethodNotAllowed:
			_ = app.methodNotAllowedResponse(c)
			return
		default:
			_ = app.errorResponse(c, httpErr.Code, httpErr.Message)
			return
		}
	}

	_ = app.serverErrorResponse(c, err)
}
