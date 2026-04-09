package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func (app *application) serve() error {
	e := app.routes()
	addr := fmt.Sprintf(":%d", app.config.port)

	// Keep a small amount of server config while still using Echo-native start.
	e.Server.ReadTimeout = 5 * time.Second
	e.Server.WriteTimeout = 10 * time.Second
	e.Server.IdleTimeout = time.Minute

	shutdownError := make(chan error, 1)

	// Graceful shutdown: stop the listener and then wait for background jobs.
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.logger.Info("shutting down server", "signal", s.String())

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err := e.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
			return
		}

		app.logger.Info("completing background tasks")
		app.wg.Wait()
		shutdownError <- nil
	}()

	app.logger.Info("starting server", "addr", addr, "env", app.config.env)

	err := e.Start(addr)
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	app.logger.Info("stopped server", "addr", addr)
	return nil
}
