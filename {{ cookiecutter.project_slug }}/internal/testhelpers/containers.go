package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"time"

	"{{ cookiecutter.go_module_path.strip() }}/assets"
	"{{ cookiecutter.go_module_path.strip() }}/internal/store"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var TestLogger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	Level: slog.LevelDebug,
}))

type PostgresContainer struct {
	*postgres.PostgresContainer
	Pool             *pgxpool.Pool
	Queries          *store.Queries
	ConnectionString string
}

func CreatePostgresContainer(ctx context.Context) (*PostgresContainer, error) {
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.WithSQLDriver("pgx"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		return nil, err
	}
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	if err := runMigrations(connStr); err != nil {
		return nil, err
	}

	if err := loadTestData(ctx, connStr); err != nil {
		return nil, err
	}
	// todo: consider removing this if not used long term
	if err := pgContainer.Snapshot(ctx, postgres.WithSnapshotName("test-snap")); err != nil {
		return nil, err
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}
	poolConfig.MaxConnLifetime = 3 * time.Minute
	poolConfig.MaxConnIdleTime = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}
	queries := store.New(pool)
	return &PostgresContainer{
		PostgresContainer: pgContainer,
		ConnectionString:  connStr,
		Pool:              pool,
		Queries:           queries,
	}, nil
}

func runMigrations(connStr string) error {
	// Get a connection from the pool
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database for migrations: %w", err)
	}
	defer db.Close()

	// Set up goose with our embedded migrations
	goose.SetBaseFS(assets.EmbeddedAssets)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Run migrations
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func loadTestData(ctx context.Context, connStr string) error {
	// Read the test data SQL file
	possiblePaths := []string{
		"sql/tests/test-data.sql",       // From project root
		"../sql/tests/test-data.sql",    // From internal directory
		"../../sql/tests/test-data.sql", // From internal/testhelpers
	}

	var testData []byte
	var err error
	var foundPath string

	for _, path := range possiblePaths {
		testData, err = os.ReadFile(path)
		if err == nil {
			foundPath = path
			break
		}
	}

	if err != nil {
		// If we still can't find it, try using the current working directory
		cwd, _ := os.Getwd()
		slog.Info("Searching for test data file", "cwd", cwd)
		return fmt.Errorf("failed to read test data file from any path: %w", err)
	}

	slog.Info("Found test data file", "path", foundPath)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database for loading test data: %w", err)
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, string(testData))
	if err != nil {
		return fmt.Errorf("failed to execute test data SQL: %w", err)
	}
	return nil
}

// Close cleans up the container and database resources
func (pc *PostgresContainer) Close(ctx context.Context) {
	if pc.Pool != nil {
		pc.Pool.Close()
	}

	if pc.PostgresContainer != nil {
		if err := pc.PostgresContainer.Terminate(ctx); err != nil {
			slog.Error("failed to terminate container", "err", err)
		}
	}
}
