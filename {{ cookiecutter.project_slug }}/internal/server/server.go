package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/metrics"
{% if cookiecutter.use_river -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/jobs"
{% endif -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"

{% if not cookiecutter.api_only -%}
	"github.com/alexedwards/scs/v2"
{% if cookiecutter.database_choice == 'postgres' -%}
	"github.com/alexedwards/scs/pgxstore"
{% endif -%}
{% endif -%}
{% if cookiecutter.database_choice == 'postgres' -%}
	"github.com/jackc/pgx/v5/pgxpool"
{% endif -%}
{% if cookiecutter.use_nats -%}
	"github.com/nats-io/nats.go"
{% endif -%}
	"golang.org/x/sync/errgroup"
)

// shutdownGrace bounds how long in-flight requests have to finish once a
// signal arrives.
const shutdownGrace = 10 * time.Second

// Deps is everything the server needs. A struct rather than positional
// parameters so adding a dependency does not change every call site.
type Deps struct {
	Conf *config.Conf
	Log  *slog.Logger
	Db   *store.Queries
{% if cookiecutter.database_choice == 'postgres' -%}
	PgxPool *pgxpool.Pool
{% endif -%}
{% if cookiecutter.use_nats -%}
	Nats *nats.Conn
{% endif -%}
{% if cookiecutter.use_river -%}
	Jobs *jobs.Client
{% endif -%}
}

type App struct {
	Deps
	Metrics *metrics.Metrics
{% if not cookiecutter.api_only -%}
	Sessions *scs.SessionManager
	csrf     *http.CrossOriginProtection
{% endif -%}
}

func New(d Deps) *App {
	app := &App{Deps: d, Metrics: metrics.New()}
{% if not cookiecutter.api_only -%}
	app.Sessions = newSessionManager(d)
	app.csrf = newCrossOriginProtection(d.Conf)
{% endif -%}
	return app
}

{% if not cookiecutter.api_only -%}
func newSessionManager(d Deps) *scs.SessionManager {
	s := scs.New()
	s.Lifetime = d.Conf.Session.Lifetime
	s.Cookie.Secure = d.Conf.Session.Secure
	s.Cookie.HttpOnly = true
	s.Cookie.SameSite = http.SameSiteLaxMode
{% if cookiecutter.database_choice == 'postgres' -%}
	s.Store = pgxstore.New(d.PgxPool)
{% else -%}
	// Left on the default in-memory store: scs has no CGO-free SQLite store,
	// so sessions do not survive a restart. Swap it if that matters.
{% endif -%}
	return s
}

// newCrossOriginProtection builds the stdlib CSRF check. A rejected request is
// answered with 404 rather than 403 so a probe cannot use the status code to
// confirm that a resource exists.
//
// The panic cannot fire from configuration: Load builds this from the same
// list through the same function, so reaching it means the two have drifted
// apart.
func newCrossOriginProtection(cfg *config.Conf) *http.CrossOriginProtection {
	p, problems := config.NewCrossOriginProtection(cfg.AppConf.TrustedOrigins)
	if len(problems) > 0 {
		panic(fmt.Sprintf("server: %v", problems))
	}
	p.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	return p
}
{% endif -%}

func (app *App) Start(ctx context.Context) error {
{% if cookiecutter.use_river -%}
	return app.Jobs.Start(ctx)
{% else -%}
	return nil
{% endif -%}
}

func (app *App) Stop(ctx context.Context) error {
{% if cookiecutter.use_river -%}
	return app.Jobs.Stop(ctx)
{% else -%}
	return nil
{% endif -%}
}

// newHTTPServer builds a listener on port carrying the configured timeouts.
func (app *App) newHTTPServer(port int, h http.Handler) *http.Server {
	return &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      h,
		IdleTimeout:  app.Conf.Server.TimeoutIdle,
		ReadTimeout:  app.Conf.Server.TimeoutRead,
		WriteTimeout: app.Conf.Server.TimeoutWrite,
	}
}

// listen serves srv until it is shut down. ErrServerClosed is what Shutdown
// leaves behind, not a failure.
func (app *App) listen(name string, srv *http.Server) func() error {
	return func() error {
		app.Log.Info(name+" listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// Serve runs the HTTP server until a signal arrives or a goroutine fails,
// then drains in-flight requests within shutdownGrace.
func (app *App) Serve(ctx context.Context) error {
	srv := app.newHTTPServer(app.Conf.Server.Port, app.Routes())
	// A listener of its own, on a port nothing publishes: process statistics
	// and the route table are worth handing an operator and no one else.
	metricsSrv := app.newHTTPServer(app.Conf.Server.MetricsPort, app.Metrics.Handler())

	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	g, gctx := errgroup.WithContext(ctx)

	g.Go(app.listen("HTTP server", srv))
	g.Go(app.listen("metrics server", metricsSrv))

	g.Go(func() error {
		<-gctx.Done()
		app.Log.Warn("shutting down", "addr", srv.Addr)
		// WithoutCancel: gctx is already done by the time we get here, so a
		// timeout derived from it would expire immediately and cut off the
		// requests this grace period exists to drain.
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownGrace)
		defer cancel()

		// Together, not in turn: Shutdown blocks until its own listener has
		// drained, so a metrics scrape that hangs would otherwise spend the
		// grace period that in-flight requests exist to use.
		var draining errgroup.Group
		draining.Go(func() error {
			// Scrapes are short, so the metrics listener has nothing to drain
			// and its failure to close is not worth failing the shutdown over.
			if err := metricsSrv.Shutdown(shutdownCtx); err != nil {
				app.Log.Error("metrics server shutdown", "error", err)
			}
			return nil
		})
		draining.Go(func() error { return srv.Shutdown(shutdownCtx) })
		return draining.Wait()
	})

	// A signal is the ordinary way to stop, so its cancellation is not an error.
	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	app.Log.Info("server stopped")
	return nil
}
