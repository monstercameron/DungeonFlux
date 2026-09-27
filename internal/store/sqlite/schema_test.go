package sqlite

import (
	"context"
	"database/sql"
	"testing"
)

func TestApplyMigrations_isIdempotentAndCreatesTables(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("migration count = %d, want 1", count)
	}
	for _, table := range []string{"runs", "events", "seats", "characters", "assets", "cache", "recordings"} {
		var exists int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if exists != 1 {
			t.Fatalf("table %q was not created", table)
		}
	}
}

func TestApplyMigrations_rejectsNilDatabase(t *testing.T) {
	if err := ApplyMigrations(context.Background(), nil); err == nil {
		t.Fatal("expected nil database error")
	}
}

func TestApplyMigrations_honorsCanceledContext(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := ApplyMigrations(ctx, db); err == nil {
		t.Fatal("expected canceled context error")
	}
}
