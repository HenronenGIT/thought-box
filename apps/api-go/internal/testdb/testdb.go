// Package testdb provides per-test Postgres pools backed by a real database.
//
// Tests opt in by calling Pool(t). If the test database is unreachable the
// test is skipped, so unit-test runs without Docker remain green.
package testdb

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/HenronenGIT/thought-box/apps/api-go/internal/migrations"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultURL points at the dedicated test database in docker-compose.
const DefaultURL = "postgres://thoughts:thoughts@localhost:5432/thoughts_test?sslmode=disable&default_query_exec_mode=simple_protocol"

var (
	migrateOnce sync.Once
	migrateErr  error
)

// Pool returns a connection pool to the test database. If the database is
// not reachable the test is skipped. Migrations run exactly once per process.
func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = DefaultURL
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("test database unavailable: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("test database ping failed (is docker-compose up?): %v", err)
	}

	migrateOnce.Do(func() {
		migrateErr = migrations.Up(ctx, url)
	})
	if migrateErr != nil {
		pool.Close()
		t.Fatalf("test database migration failed: %v", migrateErr)
	}

	t.Cleanup(func() { pool.Close() })
	return pool
}

// Truncate empties the given tables. Call at the start of a test that needs
// a clean slate. Order matters where foreign keys exist: pass children first.
func Truncate(t *testing.T, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	ctx := context.Background()
	for _, table := range tables {
		if _, err := pool.Exec(ctx, "truncate table "+table+" cascade"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}
