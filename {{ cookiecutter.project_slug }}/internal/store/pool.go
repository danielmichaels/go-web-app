package store

import (
	"context"
{% if cookiecutter.database_choice == 'postgres' -%}
	"errors"
	"fmt"
{% endif -%}
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

{% if cookiecutter.database_choice == 'postgres' -%}
	"github.com/jackc/pgx/v5/pgxpool"
{% else -%}
	"database/sql"

	_ "modernc.org/sqlite"
{% endif -%}
)

{% if cookiecutter.database_choice == 'postgres' -%}
// ErrNoDSN is returned when nothing supplied a database URL — neither
// DATABASE_URL nor an embedded instance writing its own DSN back into config.
var ErrNoDSN = errors.New("store: no database URL configured")

const minConns = 2

// NewDatabasePool opens the pool against cfg.Db.URL. By this point the URL is
// either the deployment's DATABASE_URL or the DSN the embedded instance wrote
// back into config at boot, so there is only one code path here.
func NewDatabasePool(ctx context.Context, cfg *config.Conf) (*pgxpool.Pool, error) {
	if cfg.Db.URL == "" {
		return nil, ErrNoDSN
	}
	// Pool sizing is set on the parsed config rather than appended to the DSN:
	// a supplied URL may carry no query string at all, and concatenating
	// "&pool_max_conns=..." onto it silently corrupts the database name.
	c, err := pgxpool.ParseConfig(cfg.Db.URL)
	if err != nil {
		return nil, fmt.Errorf("store: parse database url: %w", err)
	}
	c.MaxConns = int32(cfg.Db.MaxConns)
	c.MinConns = minConns
	c.MaxConnLifetime = time.Hour
	c.MaxConnIdleTime = 30 * time.Second

	return pgxpool.NewWithConfig(ctx, c)
}
{% else -%}
// DSN turns a database file path into a SQLite connection string with the
// pragmas required by the application's single-writer and Litestream model.
// busy_timeout waits for the current writer instead of returning SQLITE_BUSY,
// WAL permits readers during writes and is required for Litestream, and
// foreign_keys makes SQLite enforce the same relational constraints callers
// expect from the schema.
func DSN(path string) string {
	return path + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)"
}

func NewDatabasePool(ctx context.Context, cfg *config.Conf) (*sql.DB, error) {
	db, err := sql.Open("sqlite", DSN(cfg.Db.DbName))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
{% endif -%}
