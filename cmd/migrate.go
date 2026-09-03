package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"text/tabwriter"
	"time"

	// Registers the "pgx" database/sql driver that is used by goose.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/paveltessman/yaa/migrations"
	"github.com/paveltessman/yaa/platform/settings"
)

const migrateDriver = "pgx"

var migrateActions = map[string]func(context.Context, *goose.Provider) error{
	"up":      migrateUp,
	"down":    migrateDown,
	"status":  migrateStatus,
	"version": migrateVersion,
}

func migrate(ctx context.Context, args []string) error {
	return runMigrate(ctx, settings.DatabaseURL(), args)
}

func runMigrate(ctx context.Context, databaseURL string, args []string) error {
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	run, ok := migrateActions[action]
	if !ok {
		return fmt.Errorf("migrate: unknown action %q, want up, down, status or version", action)
	}

	sqlFiles, err := migrations.Migrations()
	if err != nil {
		return err
	}

	db, err := sql.Open(migrateDriver, databaseURL)
	if err != nil {
		return fmt.Errorf("migrate: can't open the database: %w", err)
	}
	defer func() { _ = db.Close() }()

	provider, err := goose.NewProvider(goose.DialectPostgres, db, sqlFiles)
	if err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return run(ctx, provider)
}

func migrateUp(ctx context.Context, provider *goose.Provider) error {
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if len(results) == 0 {
		log.Println("schema is up to date")
		return nil
	}
	for _, r := range results {
		log.Printf("applied %s", r.String())
	}
	return nil
}

func migrateDown(ctx context.Context, provider *goose.Provider) error {
	result, err := provider.Down(ctx)
	if err != nil {
		if errors.Is(err, goose.ErrNoNextVersion) {
			log.Println("nothing to roll back")
			return nil
		}
		return fmt.Errorf("migrate down: %w", err)
	}
	log.Printf("rolled back %s", result.String())
	return nil
}

func migrateVersion(ctx context.Context, provider *goose.Provider) error {
	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return fmt.Errorf("migrate version: %w", err)
	}
	log.Printf("schema version %d", version)
	return nil
}

func migrateStatus(ctx context.Context, provider *goose.Provider) error {
	statuses, err := provider.Status(ctx)
	if err != nil {
		return fmt.Errorf("migrate status: %w", err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	_, _ = fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tSOURCE")
	for _, s := range statuses {
		appliedAt := "-"
		if !s.AppliedAt.IsZero() {
			appliedAt = s.AppliedAt.Format(time.RFC3339)
		}
		_, _ = fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Source.Version, s.State, appliedAt, s.Source.Path)
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("migrate status: %w", err)
	}
	return nil
}
