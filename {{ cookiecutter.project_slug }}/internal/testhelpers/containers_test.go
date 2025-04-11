package testhelpers

import (
	"context"
	"testing"
	"time"
)

func TestPostgresContainer(t *testing.T) {
	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Create the Postgres container
	pgContainer, err := CreatePostgresContainer(ctx)
	if err != nil {
		t.Fatalf("failed to create postgres container: %v", err)
	}
	// Ensure container is terminated after test
	defer pgContainer.Close(ctx)
	var count int
	err = pgContainer.Pool.QueryRow(ctx, "SELECT 1").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query database: %v", err)
	}
	t.Logf("Successfully connected to test database")
}
