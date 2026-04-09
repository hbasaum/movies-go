package main

import (
	"errors"
	"strings"
	"time"

	"github.com/hbasaum/lfg-echo/internal/data"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

func (app *application) rateLimit() echo.MiddlewareFunc {
	// Echo's in-memory limiter handles per-identifier buckets and cleanup for us.
	store := echomw.NewRateLimiterMemoryStoreWithConfig(echomw.RateLimiterMemoryStoreConfig{
		Rate:      rate.Limit(app.config.limiter.rps),
		Burst:     app.config.limiter.burst,
		ExpiresIn: 3 * time.Minute,
	})

	return echomw.RateLimiterWithConfig(echomw.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		DenyHandler: func(c echo.Context, _ string, _ error) error {
			return app.rateLimitExceededResponse(c)
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return app.serverErrorResponse(c, err)
		},
	})
}

func (app *application) authenticate(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Add("Vary", "Authorization")

		authorizationHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authorizationHeader == "" {
			app.contextSetUser(c, data.AnonymousUser)
			return next(c)
		}

		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			return app.invalidCredentialsResponse(c)
		}

		token := headerParts[1]
		if token == "" || len(token) != 26 {
			return app.invalidCredentialsResponse(c)
		}

		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				return app.invalidAuthenticationTokenResponse(c)
			default:
				return app.serverErrorResponse(c, err)
			}
		}

		app.contextSetUser(c, user)
		return next(c)
	}
}

func (app *application) requireAuthenticatedUser() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := app.contextGetUser(c)
			if user.IsAnonymous() {
				return app.authenticationRequiredResponse(c)
			}
			return next(c)
		}
	}
}

func (app *application) requireActivatedUser() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := app.contextGetUser(c)

			if user.IsAnonymous() {
				return app.authenticationRequiredResponse(c)
			}
			if !user.Activated {
				return app.inactiveAccountResponse(c)
			}

			return next(c)
		}
	}
}

func (app *application) requirePermission(code string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			user := app.contextGetUser(c)

			permissions, err := app.models.Permissions.GetAllForUser(user.ID)
			if err != nil {
				return app.serverErrorResponse(c, err)
			}

			if !permissions.Include(code) {
				return app.notPermittedResopnse(c)
			}

			return next(c)
		}
	}
}
