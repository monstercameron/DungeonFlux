// Package sqlite provides SQLite-backed implementations of DungeonFlux storage ports.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

const schemaVersion = 1

// ApplyMigrations creates the storage schema if it does not already exist.
// Migrations are safe to run repeatedly against the same database.
func ApplyMigrations(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("apply sqlite migrations: nil database")
	}
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema migrations: %w", err)
	}
	var applied bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, schemaVersion).Scan(&applied); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if applied {
		return nil
	}
	if err := applyVersionOne(ctx, db); err != nil {
		return err
	}
	return nil
}

func applyVersionOne(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS runs (
			id TEXT PRIMARY KEY,
			room TEXT NOT NULL,
			started_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			mode TEXT NOT NULL,
			seed BLOB NOT NULL,
			features TEXT NOT NULL DEFAULT '{}',
			config_hash TEXT NOT NULL DEFAULT '',
			run_index INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			seq INTEGER NOT NULL,
			at_ms INTEGER NOT NULL DEFAULT 0,
			kind TEXT NOT NULL,
			machine TEXT NOT NULL DEFAULT '',
			epoch INTEGER NOT NULL DEFAULT 0,
			scope_key TEXT NOT NULL DEFAULT '',
			from_state TEXT NOT NULL DEFAULT '',
			event_json BLOB,
			note_json BLOB,
			to_state TEXT NOT NULL DEFAULT '',
			effects_json BLOB,
			outputs_json BLOB,
			PRIMARY KEY (run_id, seq)
		)`,
		`CREATE INDEX IF NOT EXISTS events_run_seq ON events(run_id, seq)`,
		`CREATE TABLE IF NOT EXISTS seats (
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			seat INTEGER NOT NULL,
			player_number INTEGER NOT NULL,
			token TEXT NOT NULL DEFAULT '',
			room_code TEXT NOT NULL DEFAULT '',
			host_token TEXT NOT NULL DEFAULT '',
			dm_token TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (run_id, seat)
		)`,
		`CREATE TABLE IF NOT EXISTS characters (
			run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
			id TEXT NOT NULL,
			seat INTEGER NOT NULL DEFAULT 0,
			character_json BLOB NOT NULL,
			PRIMARY KEY (run_id, id)
		)`,
		`CREATE TABLE IF NOT EXISTS assets (
			sha256 TEXT PRIMARY KEY,
			id TEXT NOT NULL,
			kind TEXT NOT NULL,
			mime TEXT NOT NULL,
			path TEXT NOT NULL DEFAULT '',
			duration_ms INTEGER NOT NULL DEFAULT 0,
			size INTEGER NOT NULL DEFAULT 0,
			input_hash TEXT NOT NULL DEFAULT '',
			meta_json BLOB
		)`,
		`CREATE TABLE IF NOT EXISTS cache (
			adapter TEXT NOT NULL,
			input_hash TEXT NOT NULL,
			response BLOB NOT NULL,
			created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (adapter, input_hash)
		)`,
		`CREATE TABLE IF NOT EXISTS recordings (
			adapter TEXT NOT NULL,
			phase TEXT NOT NULL,
			seat INTEGER NOT NULL,
			call_index INTEGER NOT NULL,
			recording_id TEXT NOT NULL,
			text TEXT NOT NULL DEFAULT '',
			audio BLOB,
			mime TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (adapter, phase, seat, call_index)
		)`,
	}
	for _, statement := range statements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply schema statement: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, schemaVersion); err != nil {
		return fmt.Errorf("record schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit schema migration: %w", err)
	}
	return nil
}
