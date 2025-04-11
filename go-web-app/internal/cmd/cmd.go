package cmd

import (
	"context"
	"github.com/danielmichaels/go-web-app/internal/config"
	"github.com/danielmichaels/go-web-app/internal/logging"
	"github.com/danielmichaels/go-web-app/internal/store"
	"log/slog"
	"github.com/jackc/pgx/v5/pgxpool"
	
)

type Globals struct {
}

type Setup struct {
	Config  *config.Conf
	Logger  *slog.Logger
	PgxPool *pgxpool.Pool
	
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

	if err := db.Ping(ctx); err != nil {
		logger.Error("database ping error", "error", err)
		db.Close()
		cancel()
		return nil, err
	}
	
	s := &Setup{
		Config:  cfg,
		Logger:  logger,
	PgxPool: db,
	
		Store:   store.New(db),
		Ctx:     ctx,
		Cancel:  cancel,
	}
	return s, nil
}

func (s *Setup) Close() {
	s.Logger.Info("shutting down PgxPool")
	s.Cancel()
	s.PgxPool.Close()
	
	s.Logger.Info("shutdown complete")
}
