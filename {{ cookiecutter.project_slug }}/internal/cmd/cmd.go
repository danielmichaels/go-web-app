package cmd

import (
	"context"
	"log/slog"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/logging"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	{% if cookiecutter.database_choice == "postgres" -%}
	"github.com/jackc/pgx/v5/pgxpool"
	{% endif %}
)

type Globals struct {
}

type App struct {
	Config  *config.Conf
	Logger  *slog.Logger
	{% if cookiecutter.database_choice == "postgres" -%}
	PgxPool *pgxpool.Pool
	{% endif %}
	Store   *store.Queries
	Ctx     context.Context
	Cancel  context.CancelFunc
}

func NewApp() (*App, error) {
	cfg := config.AppConfig()
	logger := logging.SetupLogger(cfg)
	ctx, cancel := context.WithCancel(context.Background())

	db, err := store.NewDatabasePool(ctx, cfg)
	if err != nil {
		logger.Error("database error", "error", err)
		cancel()
		return nil, err
	}
{% if cookiecutter.database_choice == "postgres" %}
	if err := db.Ping(ctx); err != nil {
		logger.Error("database ping error", "error", err)
		db.Close()
		cancel()
		return nil, err
	}
	{% endif %}
	a := &App{
		Config:  cfg,
		Logger:  logger,
	{% if cookiecutter.database_choice == "postgres" -%}
		PgxPool: db,
	{% endif %}
		Store:   store.New(db),
		Ctx:     ctx,
		Cancel:  cancel,
	}
	return a, nil
}

func (a *App) Close() {
	a.Logger.Info("shutting down")
	a.Cancel()
	{% if cookiecutter.database_choice == "postgres" -%}
	a.PgxPool.Close()
	{% endif %}
	a.Logger.Info("shutdown complete")
}
