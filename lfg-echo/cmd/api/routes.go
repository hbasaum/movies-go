package main

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func (app *application) routes() *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Validator = newRequestValidator()

	// Keep framework wiring in one place so handlers can stay focused on
	// business logic (bind, validate, call model, respond).
	e.HTTPErrorHandler = app.httpErrorHandler

	// Middleware order matters in Echo: each layer wraps the next one.
	e.Use(middleware.Recover())

	if len(app.config.cors.trustedOrigins) > 0 {
		e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
			AllowOrigins: app.config.cors.trustedOrigins,
			AllowMethods: []string{http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete},
			AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
		}))
	}

	if app.config.limiter.enabled {
		e.Use(app.rateLimit())
	}

	// Authentication is global so every request has a user in context
	// (anonymous or authenticated), which keeps downstream checks simple.
	e.Use(app.authenticate)

	v1 := e.Group("/v1")

	v1.GET("/healthcheck", app.healthcheckHandler)

	v1.POST("/users", app.registerUserHanlder)
	v1.PUT("/users/activated", app.activateUserHandler)
	v1.POST("/tokens/authentication", app.createAuthenticationTokenHandler)

	v1.GET("/movies", app.listMoviesHandler, app.requireActivatedUser(), app.requirePermission("movies:read"))
	v1.POST("/movies", app.createMovieHandler, app.requireActivatedUser(), app.requirePermission("movies:write"))
	v1.GET("/movies/:id", app.showMovieHandler, app.requireActivatedUser(), app.requirePermission("movies:read"))
	v1.PATCH("/movies/:id", app.updateMovieHandler, app.requireActivatedUser(), app.requirePermission("movies:write"))
	v1.DELETE("/movies/:id", app.deleteMovieHandler, app.requireActivatedUser(), app.requirePermission("movies:write"))

	return e
}
