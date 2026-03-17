package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	{% if cookiecutter.database_choice == "postgres" -%}
	"github.com/jackc/pgx/v5/pgxpool"
	{% endif %}
	{% if cookiecutter.use_nats -%}
	"github.com/nats-io/nats.go"
	{% endif %}
	{% if cookiecutter.use_river -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/jobs"
	{% endif %}
)

type App struct {
	Conf    *config.Conf
	Log     *slog.Logger
	Db      *store.Queries
	{% if cookiecutter.database_choice == "postgres" -%}
	PgxPool *pgxpool.Pool
	{% endif -%}
	{% if cookiecutter.use_nats -%}
	Nats *nats.Conn
	{% endif -%}
	{% if cookiecutter.use_river -%}
	Jobs *jobs.Client
	{% endif %}
}

func New(
	c *config.Conf,
	l *slog.Logger,
	{% if cookiecutter.database_choice == "postgres" -%}
	db *store.Queries,
	pgxPool *pgxpool.Pool,
	{% endif -%}
	{% if cookiecutter.use_nats -%}
	n *nats.Conn,
	{% endif -%}
	{% if cookiecutter.use_river -%}
	j *jobs.Client,
	{% endif %}
) *App {
	return &App{
		Conf: c,
		Log:  l,
	{% if cookiecutter.database_choice == "postgres" -%}
		Db:      db,
		PgxPool: pgxPool,
	{% endif -%}
	{% if cookiecutter.use_nats -%}
		Nats: n,
	{% endif -%}
	{% if cookiecutter.use_river -%}
		Jobs: j,
	{% endif -%}
	}
}

func (app *App) Start(ctx context.Context) error {
	{% if cookiecutter.use_river -%}
	return app.Jobs.Start(ctx)
	{% else -%}
	return nil
	{% endif %}
}

func (app *App) Stop(ctx context.Context) error {
	{% if cookiecutter.use_river -%}
	return app.Jobs.Stop(ctx)
	{% else -%}
	return nil
	{% endif %}
}

func (app *App) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", app.Conf.Server.Port),
		Handler:      app.routes(),
		IdleTimeout:  app.Conf.Server.TimeoutIdle,
		ReadTimeout:  app.Conf.Server.TimeoutRead,
		WriteTimeout: app.Conf.Server.TimeoutWrite,
	}
	app.Log.Info("HTTP server listening", "port", app.Conf.Server.Port)
	wg := sync.WaitGroup{}
	shutdownError := make(chan error)
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		app.Log.Warn("signal caught", "signal", s.String())

		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		app.Log.Warn("web-server", "addr", srv.Addr, "msg", "completing background tasks")
		wg.Wait()
		shutdownError <- nil
	}()

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	err = <-shutdownError
	if err != nil {
		app.Log.Warn("web-server shutdown err", "addr", srv.Addr, "msg", "stopped server")
		return err
	}
	return nil
}
