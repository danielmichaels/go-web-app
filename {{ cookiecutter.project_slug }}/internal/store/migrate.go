package store

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
{% if cookiecutter.database_choice == 'sqlite' -%}
	"os"
	"path/filepath"
{% endif -%}
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/assets"

{% if cookiecutter.database_choice == 'postgres' -%}
	_ "github.com/jackc/pgx/v5/stdlib"
{% else -%}
	_ "modernc.org/sqlite"
{% endif -%}
	"github.com/pressly/goose/v3"
)

const (
	// migrationsDir is the path inside assets.EmbeddedFiles, not on disk:
	// migrations ship in the binary so no host needs the goose CLI.
	migrationsDir = "migrations"

	pingAttempts = 30
	pingInterval = time.Second
)

// MigrateUp applies every pending migration. Every replica runs this at boot.
{% if cookiecutter.database_choice == 'postgres' -%}
// Concurrent runs are serialised with an advisory lock — goose is not safe to
// race, as parallel runs fight over the version table. The replica that
// arrives second waits, then finds nothing to do.
{% endif -%}
//
// A nil logger discards goose output.
func MigrateUp(ctx context.Context, dsn string, logger *slog.Logger) error {
	return migrate(ctx, dsn, logger, func(db *sql.DB) error {
		return goose.Up(db, migrationsDir)
	})
}

// MigrateUpTo applies migrations up to and including version, leaving later
// ones pending. Tests use it to exercise a specific pre-migration schema.
func MigrateUpTo(ctx context.Context, dsn string, version int64, logger *slog.Logger) error {
	return migrate(ctx, dsn, logger, func(db *sql.DB) error {
		return goose.UpTo(db, migrationsDir, version)
	})
}

func migrate(ctx context.Context, dsn string, logger *slog.Logger, fn func(*sql.DB) error) error {
	db, err := prepareMigrationDB(ctx, dsn, logger)
	if err != nil {
		return err
	}
	defer db.Close()

	goose.SetBaseFS(assets.EmbeddedFiles)
	if err := goose.SetDialect("{% if cookiecutter.database_choice == 'postgres' %}postgres{% else %}sqlite3{% endif %}"); err != nil {
		return fmt.Errorf("store: set goose dialect: %w", err)
	}
	if logger == nil {
		goose.SetLogger(goose.NopLogger())
	}

{% if cookiecutter.database_choice == 'postgres' -%}
	if err := WithAdvisoryLock(ctx, db, AdvisoryLockMigration, func(context.Context) error {
		return fn(db)
	}); err != nil {
		return fmt.Errorf("store: run migrations: %w", err)
	}
{% else -%}
	// No advisory lock: SQLite is single-writer and single-node by nature.
	if err := fn(db); err != nil {
		return fmt.Errorf("store: run migrations: %w", err)
	}
{% endif -%}
	return nil
}

// prepareMigrationDB opens a migration connection and waits for the database
// to accept queries. The wait matters on boot: an embedded Postgres or a
// container started alongside this process may not be listening yet.
func prepareMigrationDB(ctx context.Context, dsn string, logger *slog.Logger) (*sql.DB, error) {
{% if cookiecutter.database_choice == 'sqlite' -%}
	// SQLite will not create missing parent directories, and the directory is
	// absent both in a fresh checkout and in the container image, where it is
	// excluded from the build context.
	if dir := filepath.Dir(dsn); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("store: create database directory %s: %w", dir, err)
		}
	}

{% endif -%}
	db, err := sql.Open("{% if cookiecutter.database_choice == 'postgres' %}pgx{% else %}sqlite{% endif %}", {% if cookiecutter.database_choice == 'postgres' %}dsn{% else %}DSN(dsn){% endif %})
	if err != nil {
		return nil, fmt.Errorf("store: open database for migrations: %w", err)
	}

	var lastErr error
	for attempt := 1; attempt <= pingAttempts; attempt++ {
		if lastErr = db.PingContext(ctx); lastErr == nil {
			return db, nil
		}
		if ctx.Err() != nil {
			db.Close()
			return nil, fmt.Errorf("store: waiting for database: %w", ctx.Err())
		}
		if logger != nil {
			logger.Debug(
				"waiting for database",
				"attempt", attempt,
				"of", pingAttempts,
				"err", lastErr,
			)
		}
		time.Sleep(pingInterval)
	}

	db.Close()
	return nil, fmt.Errorf("store: database unreachable after %d attempts: %w", pingAttempts, lastErr)
}
