package server

import (
	"context"
	"errors"
	"fmt"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/smtp"
	"{{ cookiecutter.go_module_path.strip() }}/internal/repository"
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Application struct {
	Config *config.Conf
	Logger zerolog.Logger
	wg     sync.WaitGroup
	Mailer *smtp.Mailer
	Db     *repository.Queries
}

func (app *Application) Serve() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.Config.Server.Port),
		Handler:      app.routes(),
		IdleTimeout:  app.Config.Server.TimeoutIdle,
		ReadTimeout:  app.Config.Server.TimeoutRead,
		WriteTimeout: app.Config.Server.TimeoutWrite,
	}

	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.Logger.Warn().Str("signal", s.String()).Msg("caught signal")

		// Allow processes to finish with a ten-second window
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		app.Logger.Warn().Str("tasks", srv.Addr).Msg("completing background tasks")
		// Call wait so that the wait group can decrement to zero.
		app.wg.Wait()
		shutdownError <- nil
	}()
	app.Logger.Info().Str("server", srv.Addr).Msg("starting server")

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	err = <-shutdownError
	if err != nil {
		app.Logger.Warn().Str("server", srv.Addr).Msg("stopped server")
		return err
	}
	return nil
}
