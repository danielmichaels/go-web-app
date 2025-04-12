package server

import (
	"context"
	"errors"
	"fmt"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	"{{ cookiecutter.go_module_path.strip() }}/internal/version"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	{% if cookiecutter.database_choice == "postgres" -%}
	"github.com/jackc/pgx/v5/pgxpool"
	{% endif %}
	{% if cookiecutter.use_nats -%}
	"github.com/nats-io/nats.go"
	{% endif %}

	"github.com/go-chi/httplog/v2"
)

type Server struct {
	Conf    *config.Conf
	Log     *slog.Logger
	Db      *store.Queries
	{% if cookiecutter.database_choice == "postgres" -%}
	PgxPool *pgxpool.Pool
	{% endif -%}
	{% if cookiecutter.use_nats -%}
	Nats *nats.Conn
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
	{% endif %}
) *Server {
	return &Server{
	Conf: c,
	Log: l,
	{% if cookiecutter.database_choice == "postgres" -%}
	Db: db,
	PgxPool: pgxPool,
	{% endif -%}
	{% if cookiecutter.use_nats -%}
	Nats: n,
	{% endif -%}
	}
}

func httpLogger(cfg *config.Conf) *httplog.Logger {
	var output io.Writer = os.Stdout
	logger := httplog.NewLogger("web", httplog.Options{
		JSON:             cfg.AppConf.LogJson,
		LogLevel:         cfg.AppConf.LogLevel,
		Concise:          cfg.AppConf.LogConcise,
		RequestHeaders:   cfg.AppConf.LogRequestHeaders,
		ResponseHeaders:  cfg.AppConf.LogResponseHeaders,
		MessageFieldName: "message",
		TimeFieldFormat:  time.RFC3339,
		Tags: map[string]string{
			"version": version.Get(),
		},
		QuietDownRoutes: []string{
			"/",
			"/ping",
			"/healthz",
		},
		QuietDownPeriod: 10 * time.Second,
		Writer:          output,
	})
	return logger
}

func (app *Server) Serve(ctx context.Context) error {
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

		// Allow processes to finish with a ten-second window
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		err := srv.Shutdown(ctx)
		if err != nil {
			shutdownError <- err
		}
		app.Log.Warn("web-server", "addr", srv.Addr, "msg", "completing background tasks")
		// Call wait so that the wait group can decrement to zero.
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
