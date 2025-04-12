package cmd

import (
	"context"
	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
	"{{ cookiecutter.go_module_path.strip() }}/internal/logging"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	"log/slog"
	{% if cookiecutter.database_choice == "postgres" -%}
	"github.com/jackc/pgx/v5/pgxpool"
	{% endif %}
)

type Globals struct {
}

type Setup struct {
	Config  *config.Conf
	Logger  *slog.Logger
	{% if cookiecutter.database_choice == "postgres" -%}
	PgxPool *pgxpool.Pool
	{% endif %}
	Store   *store.Queries
	Ctx     context.Context
	Cancel  context.CancelFunc
}

func NewSetup(service string) (*Setup, error) {
	cfg := config.AppConfig()
	logger, lctx := logging.SetupLogger(service, cfg)
	ctx, cancel := context.WithCancel(lctx)

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
	s := &Setup{
		Config:  cfg,
		Logger:  logger,
	{% if cookiecutter.database_choice == "postgres" -%}
		PgxPool: db,
	{% endif %}
		Store:   store.New(db),
		Ctx:     ctx,
		Cancel:  cancel,
	}
	return s, nil
}

func (s *Setup) Close() {
	s.Logger.Info("shutting down")
	s.Cancel()
	{% if cookiecutter.database_choice == "postgres" -%}
	s.PgxPool.Close()
	{% endif %}
	s.Logger.Info("shutdown complete")
}
