package store

import (
	"context"
	"database/sql"
	"fmt"
)

// AdvisoryLockKey namespaces the process-wide locks this application takes.
// The zero value is deliberately not a key, so a forgotten one fails loudly
// rather than silently sharing a lock with the next thing that forgets.
//
// Postgres advisory locks are a single global int64 namespace per database.
// Every key an application uses belongs in this one block, so a collision is
// visible at a glance rather than discovered as a mutual deadlock in prod.
type AdvisoryLockKey int64

const (
	// AdvisoryLockMigration serialises schema migration across replicas.
	AdvisoryLockMigration AdvisoryLockKey = iota + 1
{% if cookiecutter.use_river -%}
	// AdvisoryLockRiverMigration serialises River's own migrator, which owns
	// its queue tables and races exactly the same way goose does.
	AdvisoryLockRiverMigration
{% endif -%}
)

// advisoryLockTimeout bounds the wait for a peer. Long enough for the real
// work to finish, short enough that a wedged holder surfaces as a failure
// rather than a process hanging forever with no output.
const advisoryLockTimeout = "60s"

// WithAdvisoryLock runs fn while holding key, excluding every other session
// that asks for the same key. The lock is session-scoped rather than
// transaction-scoped because fn may need to run its own transactions — goose
// does — and so cannot itself be wrapped in one.
//
// A holder that dies releases the lock when its backend terminates, so a
// crashed peer cannot wedge the lock permanently.
func WithAdvisoryLock(
	ctx context.Context,
	db *sql.DB,
	key AdvisoryLockKey,
	fn func(context.Context) error,
) error {
	// Pinned: pg_advisory_unlock only releases a lock taken on the same
	// session, and the pool would otherwise hand the release to a different
	// connection.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("store: pin connection for advisory lock: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "SET lock_timeout = '"+advisoryLockTimeout+"'"); err != nil {
		return fmt.Errorf("store: set lock timeout: %w", err)
	}
	if _, err := conn.ExecContext(ctx, "SELECT pg_advisory_lock($1)", int64(key)); err != nil {
		return fmt.Errorf("store: acquire advisory lock %d: %w", key, err)
	}
	defer func() {
		// WithoutCancel: a cancelled boot must still release the lock now,
		// not whenever the backend eventually goes away.
		//nolint:errcheck // The lock also releases when this session ends.
		conn.ExecContext(
			context.WithoutCancel(ctx),
			"SELECT pg_advisory_unlock($1)",
			int64(key),
		)
	}()

	return fn(ctx)
}
