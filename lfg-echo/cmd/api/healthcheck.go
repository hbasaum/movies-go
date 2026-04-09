package main

import "github.com/labstack/echo/v4"

func (app *application) healthcheckHandler(c echo.Context) error {
	data := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": app.config.env,
			"version":     version,
		},
	}

	return c.JSON(200, data)
}
