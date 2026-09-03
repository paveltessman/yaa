package main

import (
	"context"
	"database/sql"
	"io/fs"
	"os"
	"testing"

	"github.com/paveltessman/yaa/migrations"
)

const (
	urlEnv = "TEST_DATABASE_URL"

	exampleURL = "postgres://yaa:yaa@localhost:5433/yaa?sslmode=disable"
)

func migrationCount(t *testing.T) int {
	t.Helper()

	sqlFiles, err := migrations.Migrations()
	if err != nil {
		t.Fatalf("Migrations: %v", err)
	}
	names, err := fs.Glob(sqlFiles, "*.sql")
	if err != nil {
		t.Fatalf("globbing: %v", err)
	}
	return len(names)
}

func rollBack(ctx context.Context, databaseURL string, count int) {
	for range count {
		_ = runMigrate(ctx, databaseURL, []string{"down"})
	}

	db, err := sql.Open(migrateDriver, databaseURL)
	if err != nil {
		return
	}
	defer func() { _ = db.Close() }()

	_, _ = db.ExecContext(ctx, `DROP TABLE IF EXISTS goose_db_version`)
}

func TestMigrateUpAndDown(t *testing.T) {
	databaseURL := os.Getenv(urlEnv)
	if databaseURL == "" {
		t.Fatalf("This test needs a database. Run `make up`, then export %s=%s", urlEnv, exampleURL)
	}
	ctx := t.Context()
	count := migrationCount(t)

	t.Cleanup(func() { rollBack(context.WithoutCancel(ctx), databaseURL, count) })
	rollBack(ctx, databaseURL, count)

	for pass := range 2 {
		for _, action := range []string{"up", "status", "version"} {
			if err := runMigrate(ctx, databaseURL, []string{action}); err != nil {
				t.Fatalf("pass %d: migrate %s: %v", pass, action, err)
			}
		}
		for range count {
			if err := runMigrate(ctx, databaseURL, []string{"down"}); err != nil {
				t.Fatalf("pass %d: migrate down: %v", pass, err)
			}
		}
	}
}

func TestMigrateRejectsAnUnknownAction(t *testing.T) {
	if err := runMigrate(t.Context(), exampleURL, []string{"sideways"}); err == nil {
		t.Error("runMigrate accepted an unknown action")
	}
}
