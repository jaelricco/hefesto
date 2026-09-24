//go:build integration

package store_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"

	"github.com/jaelricco/hefesto/internal/testutil/pgtest"
)

// Every migration must apply, roll back and re-apply cleanly: a Down that
// does not work is discovered during an incident otherwise.
func TestMigrationsRoundTrip(t *testing.T) {
	dbURL := pgtest.NewEmpty(t)
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	dir := pgtest.MigrationsDir()
	ctx := context.Background()

	if err := goose.UpContext(ctx, db, dir); err != nil {
		t.Fatalf("up: %v", err)
	}
	if err := goose.ResetContext(ctx, db, dir); err != nil {
		t.Fatalf("reset (every down): %v", err)
	}
	var tables int
	if err := db.QueryRowContext(ctx,
		`SELECT count(*) FROM pg_tables WHERE schemaname = 'public' AND tablename <> 'goose_db_version'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 0 {
		t.Fatalf("down migrations left %d tables behind", tables)
	}
	if err := goose.UpContext(ctx, db, dir); err != nil {
		t.Fatalf("up again: %v", err)
	}
}
