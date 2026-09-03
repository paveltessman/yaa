// Package dbtest runs a database test in a schema of its own.
package dbtest

import (
	"database/sql"
	"net/url"
	"os"
	"strings"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	// Registers the "pgx" database/sql driver that is used by goose.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/paveltessman/yaa/migrations"
)

const (
	driver = "pgx"

	urlEnv = "TEST_DATABASE_URL"

	exampleURL = "postgres://yaa:yaa@localhost:5433/yaa?sslmode=disable"
)

// NewPool makes a schema, applies the migrations in it, and returns a pool that
// reads and writes there. The cleanup drops the schema.
func NewPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	databaseURL := os.Getenv(urlEnv)
	if databaseURL == "" {
		t.Fatalf("This test needs a database. Run `make up`, then export %s=%s", urlEnv, exampleURL)
	}

	schema := newSchemaName()
	admin := open(t, databaseURL)
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("dbtest: can't create schema %s: %v", schema, err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec("DROP SCHEMA " + schema + " CASCADE"); err != nil {
			t.Errorf("dbtest: can't drop schema %s: %v", schema, err)
		}
		_ = admin.Close()
	})

	schemaURL := withSearchPath(t, databaseURL, schema)
	migrate(t, schemaURL)

	pool, err := pgxpool.New(t.Context(), schemaURL)
	if err != nil {
		t.Fatalf("dbtest: can't open the pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newSchemaName() string {
	return "test_" + strings.ReplaceAll(uuid.NewV4().String(), "-", "")
}

func open(t *testing.T, databaseURL string) *sql.DB {
	t.Helper()

	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		t.Fatalf("dbtest: can't open the database: %v", err)
	}
	return db
}

func withSearchPath(t *testing.T, databaseURL, schema string) string {
	t.Helper()

	parsed, err := url.Parse(databaseURL)
	if err != nil {
		t.Fatalf("dbtest: can't parse %s: %v", urlEnv, err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func migrate(t *testing.T, databaseURL string) {
	t.Helper()

	sqlFiles, err := migrations.Migrations()
	if err != nil {
		t.Fatalf("dbtest: %v", err)
	}

	db := open(t, databaseURL)
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sqlFiles)
	if err != nil {
		t.Fatalf("dbtest: %v", err)
	}
	if _, err := provider.Up(t.Context()); err != nil {
		t.Fatalf("dbtest: can't apply the migrations: %v", err)
	}
}
