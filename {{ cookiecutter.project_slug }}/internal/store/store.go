package store

import (
	"context"
	"time"
	"database/sql"

	"{{ cookiecutter.go_module_path.strip() }}/internal/config"

	_ "modernc.org/sqlite"
)
func NewDatabasePool(ctx context.Context, cfg *config.Conf) (*sql.DB, error) {
	db, err := sql.Open("sqlite", cfg.Db.DbName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}
