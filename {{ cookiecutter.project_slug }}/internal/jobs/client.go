{% if cookiecutter.use_river %}
package jobs

import (
	"context"
	"log/slog"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivermigrate"
	{% if cookiecutter.database_choice == "postgres" %}
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	{% endif %}
	{% if cookiecutter.database_choice == "sqlite" %}
	"database/sql"

	_ "modernc.org/sqlite"
	"github.com/riverqueue/river/riverdriver/riversqlite"
	{% endif %}
)

{% if cookiecutter.database_choice == "postgres" %}
type Client struct {
	River *river.Client[pgx.Tx]
}

func NewClient(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) (*Client, error) {
	workers := river.NewWorkers()
	river.AddWorker(workers, &ExampleWorker{})

	driver := riverpgxv5.New(pool)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		return nil, err
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		return nil, err
	}

	rc, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Logger:  log,
	})
	if err != nil {
		return nil, err
	}
	return &Client{River: rc}, nil
}
{% endif %}
{% if cookiecutter.database_choice == "sqlite" %}
type Client struct {
	River *river.Client[*sql.Tx]
	db    *sql.DB
}

func NewClient(ctx context.Context, dbPath string, log *slog.Logger) (*Client, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	driver := riversqlite.New(db)
	migrator, err := rivermigrate.New(driver, nil)
	if err != nil {
		db.Close()
		return nil, err
	}
	if _, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, nil); err != nil {
		db.Close()
		return nil, err
	}

	workers := river.NewWorkers()
	river.AddWorker(workers, &ExampleWorker{})

	rc, err := river.NewClient(driver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		Logger:  log,
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Client{River: rc, db: db}, nil
}
{% endif %}

func (c *Client) Start(ctx context.Context) error {
	return c.River.Start(ctx)
}

func (c *Client) Stop(ctx context.Context) error {
	return c.River.Stop(ctx)
}
{% endif %}
