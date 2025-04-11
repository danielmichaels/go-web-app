package store

import (
	"context"

	"fmt"
	"os"

	"github.com/danielmichaels/go-web-app/internal/config"
	"time"


	"github.com/jackc/pgx/v5/pgxpool"
	

)


func NewDatabasePool(ctx context.Context, cfg *config.Conf) (*pgxpool.Pool, error) {
	minConns := 2
	dbUrl := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Db.User,
		cfg.Db.Password,
		cfg.Db.Host,
		cfg.Db.Port,
		cfg.Db.Db,
		cfg.Db.SSLMode,
	)
	// fly.io exposes DATABASE_URL to all of its machines and is the recommended way to connect
	if os.Getenv("DATABASE_URL") != "" {
		dbUrl = os.Getenv("DATABASE_URL")
	}
	dbPool := fmt.Sprintf(
		"%s&pool_max_conns=%d&pool_min_conns=%d",
		dbUrl,
		cfg.Db.MaxConns,
		minConns,
	)
	c, err := pgxpool.ParseConfig(dbPool)
	if err != nil {
		return nil, err
	}

	c.MaxConnLifetime = 1 * time.Hour
	c.MaxConnIdleTime = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, c)
}


