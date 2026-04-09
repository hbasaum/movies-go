package main

import (
	"errors"
	"fmt"

	"github.com/hbasaum/lfg-echo/internal/data"
	"github.com/hbasaum/lfg-echo/internal/validator"
	"github.com/labstack/echo/v4"
)

type createMovieRequest struct {
	Title   string       `json:"title" validate:"required,max=500"`
	Year    int32        `json:"year" validate:"required,gte=1888,maxyear"`
	Runtime data.Runtime `json:"runtime" validate:"required,gt=0"`
	Genres  []string     `json:"genres" validate:"required,min=1,max=5,uniquegenres,dive,required"`
}

type updateMovieRequest struct {
	Title   *string       `json:"title" validate:"omitempty,max=500"`
	Year    *int32        `json:"year" validate:"omitempty,gte=1888,maxyear"`
	Runtime *data.Runtime `json:"runtime" validate:"omitempty,gt=0"`
	Genres  []string      `json:"genres" validate:"omitempty,min=1,max=5,uniquegenres,dive,required"`
}

func (app *application) createMovieHandler(c echo.Context) error {
	var input createMovieRequest

	// Framework-first flow: bind request body, validate tags, then execute logic.
	if err := app.bindAndValidate(c, &input); err != nil {
		return err
	}

	movie := &data.Movie{
		Title:   input.Title,
		Year:    input.Year,
		Runtime: input.Runtime,
		Genres:  input.Genres,
	}

	err := app.models.Movies.Insert(movie)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	c.Response().Header().Set("Location", fmt.Sprintf("/v1/movies/%d", movie.ID))
	return c.JSON(201, envelope{"movie": movie})
}

func (app *application) showMovieHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		return app.badRequestResponse(c, err)
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return app.notFoundResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	return c.JSON(200, envelope{"movie": movie})
}

func (app *application) updateMovieHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		return app.badRequestResponse(c, err)
	}

	movie, err := app.models.Movies.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return app.notFoundResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	var input updateMovieRequest
	if err := app.bindAndValidate(c, &input); err != nil {
		return err
	}

	if input.Title != nil {
		movie.Title = *input.Title
	}
	if input.Year != nil {
		movie.Year = *input.Year
	}
	if input.Runtime != nil {
		movie.Runtime = *input.Runtime
	}
	if input.Genres != nil {
		movie.Genres = input.Genres
	}

	v := validator.New()
	if data.ValidateMovie(v, movie); !v.Valid() {
		return app.failedValidationResponse(c, v.Errors)
	}

	err = app.models.Movies.Update(movie)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			return app.editConflictResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	return c.JSON(200, envelope{"movie": movie})
}

func (app *application) deleteMovieHandler(c echo.Context) error {
	id, err := app.readIDParam(c)
	if err != nil {
		return app.notFoundResponse(c)
	}

	err = app.models.Movies.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			return app.notFoundResponse(c)
		default:
			return app.serverErrorResponse(c, err)
		}
	}

	return c.JSON(200, envelope{"message": "movie successfully deleted"})
}

func (app *application) listMoviesHandler(c echo.Context) error {
	var input struct {
		Title  string
		Genres []string
		data.Filters
	}

	v := validator.New()
	qs := c.QueryParams()

	input.Title = app.readString(qs, "title", "")
	input.Genres = app.readCSV(qs, "genres", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "year", "runtime", "-id", "-title", "-year", "-runtime"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		return app.failedValidationResponse(c, v.Errors)
	}

	movies, metadata, err := app.models.Movies.GetAll(input.Title, input.Genres, input.Filters)
	if err != nil {
		return app.serverErrorResponse(c, err)
	}

	return c.JSON(200, envelope{"movies": movies, "metadata": metadata})
}
