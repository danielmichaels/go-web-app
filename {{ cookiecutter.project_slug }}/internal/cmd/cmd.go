package cmd

import (
	"context"
	"fmt"
	"log/slog"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
{% if cookiecutter.database_choice == 'postgres' -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/embeddedpg"
{% endif -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/logging"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
{% if cookiecutter.database_choice == 'postgres' -%}

	"github.com/jackc/pgx/v5/pgxpool"
{% endif -%}
)

type Globals struct {
}

type App struct {
	Config *config.Conf
	Logger *slog.Logger
{% if cookiecutter.database_choice == 'postgres' -%}
	PgxPool *pgxpool.Pool
	// embedded is nil whenever a DATABASE_URL was supplied.
	embedded *embeddedpg.Server
{% endif -%}
	Store  *store.Queries
	Ctx    context.Context
	Cancel context.CancelFunc
}

// NewApp loads configuration, brings up the database, and applies migrations.
// Everything it starts is released by Close, including on the error paths
// here.
func NewApp() (*App, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	logger := logging.SetupLogger(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	a := &App{
		Config: cfg,
		Logger: logger,
		Ctx:    ctx,
		Cancel: cancel,
	}

	if err := a.startDatabase(); err != nil {
		a.release()
		return nil, err
	}
	return a, nil
}

// startDatabase brings up Postgres if this process owns it, migrates, then
// opens the pool. Migrations run before the pool so a schema change is in
// place before anything queries through it.
func (a *App) startDatabase() error {
{% if cookiecutter.database_choice == 'postgres' -%}
	if config.ShouldStartEmbedded(a.Config) {
		a.Logger.Info(
			"starting embedded postgres",
			"dir", a.Config.Db.EmbeddedDir,
			"port", a.Config.Db.EmbeddedPort,
		)
		srv, err := embeddedpg.Start(embeddedpg.Options{
			Database: "{{ cookiecutter.project_slug }}",
			DataDir:  a.Config.Db.EmbeddedDir,
			Port:     uint32(a.Config.Db.EmbeddedPort),
			Logger:   a.Logger,
		})
		if err != nil {
			return fmt.Errorf("start embedded postgres: %w", err)
		}
		a.embedded = srv
		// Everything downstream reads the DSN from config and so needs no
		// knowledge of where Postgres came from.
		a.Config.Db.URL = srv.DSN
	}

	dsn := a.Config.Db.URL
{% else -%}
	dsn := a.Config.Db.DbName
{% endif -%}

	if err := store.MigrateUp(a.Ctx, dsn, a.Logger); err != nil {
		return err
	}

	db, err := store.NewDatabasePool(a.Ctx, a.Config)
	if err != nil {
		return fmt.Errorf("open database pool: %w", err)
	}
{% if cookiecutter.database_choice == 'postgres' -%}
	a.PgxPool = db
{% endif -%}
	a.Store = store.New(db)
	return nil
}

func (a *App) Close() {
	a.Logger.Info("shutting down")
	a.release()
	a.Logger.Info("shutdown complete")
}

// release tears down whatever was started, in reverse order, and tolerates
// partially-built state so NewApp can use it on its error paths.
func (a *App) release() {
	a.Cancel()
{% if cookiecutter.database_choice == 'postgres' -%}
	if a.PgxPool != nil {
		a.PgxPool.Close()
	}
	// A Stop on an adopted instance is a no-op: this process did not start it.
	if a.embedded != nil {
		if err := a.embedded.Stop(); err != nil {
			a.Logger.Error("stopping embedded postgres", "error", err)
		}
	}
{% endif -%}
}
