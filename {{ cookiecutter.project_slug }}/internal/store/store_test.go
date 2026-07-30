package store_test

import (
	"context"
	"os"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/testhelpers"
)

// One embedded Postgres per test binary. RunTestMain traps SIGINT/SIGTERM so
// an interrupted run stops it instead of orphaning the process.
//
// Copy these two lines into any package whose tests need the database.
func TestMain(m *testing.M) {
	os.Exit(testhelpers.RunTestMain(m))
}

// This is the test that proves the whole setup: no Postgres installed, no
// Docker running, no credentials configured, and real SQL still executes
// against a migrated schema.
func TestExamplesRoundTrip(t *testing.T) {
	ctx := context.Background()
	pg := testhelpers.Shared(ctx, t)
	pg.TruncateAll(ctx, t)

	rows, err := pg.Queries.ExampleSelectAll(ctx)
	if err != nil {
		t.Fatalf("select all on empty table: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("got %d rows after truncate, want 0", len(rows))
	}

	for _, text := range []string{"first", "second"} {
		if err := pg.Queries.InsertExample(ctx, text); err != nil {
			t.Fatalf("insert %q: %v", text, err)
		}
	}

	rows, err = pg.Queries.ExampleSelectAll(ctx)
	if err != nil {
		t.Fatalf("select all: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	// created_at proves the migration's default fired, not just that the
	// column exists.
	if rows[0].CreatedAt.Time.IsZero() {
		t.Error("created_at is zero; the column default did not apply")
	}
}

// TruncateAll uses RESTART IDENTITY, so ids restart from 1 in every test. A
// plain DELETE would leave the sequence advanced and any test asserting on a
// specific id would pass alone and fail in a suite.
func TestTruncateAllResetsIdentity(t *testing.T) {
	ctx := context.Background()
	pg := testhelpers.Shared(ctx, t)

	firstID := func() int32 {
		pg.TruncateAll(ctx, t)
		if err := pg.Queries.InsertExample(ctx, "only"); err != nil {
			t.Fatalf("insert: %v", err)
		}
		rows, err := pg.Queries.ExampleSelectAll(ctx)
		if err != nil {
			t.Fatalf("select all: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("got %d rows, want 1", len(rows))
		}
		return rows[0].ID
	}

	if before, after := firstID(), firstID(); before != after {
		t.Errorf("id was %d then %d; TRUNCATE should have reset the sequence", before, after)
	}
}
