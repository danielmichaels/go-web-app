{% if cookiecutter.use_river -%}
package jobs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
{% if cookiecutter.database_choice == 'postgres' or not cookiecutter.api_only -%}
	"fmt"
{% endif -%}
{% if not cookiecutter.api_only -%}
	"net/http"
{% endif -%}
	"os"
	"strings"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
{% if cookiecutter.database_choice == 'postgres' -%}
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
{% endif -%}

{% if cookiecutter.database_choice == 'postgres' -%}
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
{% else -%}
	"database/sql"

	"github.com/riverqueue/river/riverdriver/riversqlite"
	_ "modernc.org/sqlite"
{% endif -%}
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivermigrate"
{% if not cookiecutter.api_only -%}
	"riverqueue.com/riverui"
{% endif -%}
)

// maxWorkers bounds how many jobs run concurrently in this process.
const maxWorkers = 10

// clientID names this process to River. River's own default appends a
// microsecond timestamp to the hostname, which is unique but unreadable on
// every line that carries it. Uniqueness per running process is the part that
// matters — River elects its leader by this value — so the timestamp becomes
// four bytes of entropy and the host keeps only its first label.
func clientID() string {
	host, _ := os.Hostname()
	host, _, _ = strings.Cut(host, ".")
	if host == "" {
		host = "{{ cookiecutter.cmd_name.strip() }}"
	}

	var suffix [4]byte
	// Documented never to fail as of Go 1.24.
	_, _ = rand.Read(suffix[:])

	return host + "-" + hex.EncodeToString(suffix[:])
}

type Client struct {
{% if cookiecutter.database_choice == 'postgres' -%}
	River *river.Client[pgx.Tx]
{% else -%}
	River *river.Client[*sql.Tx]
	db    *sql.DB
{% endif -%}
{% if not cookiecutter.api_only -%}
	ui *riverui.Handler
{% endif -%}
}

{% if cookiecutter.database_choice == 'postgres' -%}
// NewClient builds the job client and applies River's own migrations. Those
// are separate from the application schema: River owns its queue tables.
func NewClient(
	ctx context.Context,
	pool *pgxpool.Pool,
	cfg *config.Conf,
	log *slog.Logger,
) (*Client, error) {
	driver := riverpgxv5.New(pool)
{% else -%}
// NewClient builds the job client and applies River's own migrations. Those
// are separate from the application schema: River owns its queue tables.
func NewClient(
	ctx context.Context,
	dbPath string,
	cfg *config.Conf,
	log *slog.Logger,
) (*Client, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	driver := riversqlite.New(db)
{% endif -%}

	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
{% if cookiecutter.database_choice == 'sqlite' -%}
		db.Close()
{% endif -%}
		return nil, err
	}
{% if cookiecutter.database_choice == 'postgres' -%}
	// Under the same advisory-lock discipline as the application schema.
	// River's migrator is not concurrency-safe on its own: replicas starting
	// together collide on river_migration with "already exists" and duplicate
	// key errors, and every replica but one fails to boot.
	migrationDB := stdlib.OpenDBFromPool(pool)
	defer migrationDB.Close()

	if err := store.WithAdvisoryLock(
		ctx,
		migrationDB,
		store.AdvisoryLockRiverMigration,
		func(ctx context.Context) error {
			_, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
			return err
		},
	); err != nil {
		return nil, fmt.Errorf("jobs: river migrations: %w", err)
	}
{% else -%}
	// No advisory lock: SQLite is single-writer and single-node by nature.
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		db.Close()
		return nil, err
	}
{% endif -%}

	workers := river.NewWorkers()
	river.AddWorker(workers, &ExampleWorker{})

	rc, err := river.NewClient(driver, &river.Config{
		ID: clientID(),
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: maxWorkers},
		},
		Workers: workers,
		Logger:  log,
	})
	if err != nil {
{% if cookiecutter.database_choice == 'sqlite' -%}
		db.Close()
{% endif -%}
		return nil, err
	}

{% if cookiecutter.database_choice == 'postgres' -%}
	c := &Client{River: rc}
{% else -%}
	c := &Client{River: rc, db: db}
{% endif -%}

{% if not cookiecutter.api_only -%}
	// Built only when it is going to be served: the handler keeps polling
	// caches of its own, so an unmounted one is a background cost for nothing.
	if cfg.AppConf.RiverUIEnabled {
		ui, err := riverui.NewHandler(&riverui.HandlerOpts{
			Endpoints: riverui.NewEndpoints(rc, nil),
			Logger:    log.With("component", "riverui"),
			Prefix:    cfg.AppConf.RiverUIPath,
		})
		if err != nil {
{% if cookiecutter.database_choice == 'sqlite' -%}
			db.Close()
{% endif -%}
			return nil, fmt.Errorf("jobs: build river ui: %w", err)
		}
		c.ui = ui
	}
{% endif -%}
	return c, nil
}

{% if not cookiecutter.api_only -%}
// UIHandler is the job dashboard, for mounting at config RiverUIPath, or nil
// when RIVER_UI_EMBEDDED is off. Gate it behind whatever authorisation the
// rest of the admin area uses: it can cancel and retry jobs.
func (c *Client) UIHandler() http.Handler { return c.ui }
{% endif -%}

func (c *Client) Start(ctx context.Context) error {
{% if not cookiecutter.api_only -%}
	// The dashboard keeps background caches, so it needs starting too. It is
	// nil unless RIVER_UI_EMBEDDED asked for it.
	if c.ui != nil {
		if err := c.ui.Start(ctx); err != nil {
			return err
		}
	}
{% endif -%}
	return c.River.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	return c.River.Stop(ctx)
}
{% endif -%}
