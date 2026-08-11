package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"
)

func TestSQLitePoolUsesSafeDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.db")
	t.Setenv("DATABASE_URL", path)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := NewDatabasePool(context.Background(), cfg)
	if err != nil {
		t.Fatalf("open database pool: %v", err)
	}
	defer db.Close()

	if got := db.Stats().MaxOpenConnections; got != 1 {
		t.Errorf("max open connections = %d, want 1", got)
	}

	var busyTimeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("read busy_timeout: %v", err)
	}
	if busyTimeout != 10000 {
		t.Errorf("busy_timeout = %d, want 10000", busyTimeout)
	}

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("read foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("database file was not created: %v", err)
	}
}
