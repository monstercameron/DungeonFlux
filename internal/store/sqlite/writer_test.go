package sqlite

import (
	"context"
	"database/sql"
	"log/slog"
	"testing"
)

func TestOpenClose_createsStore(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir()+"/store.db", nil)
	if err != nil {
		t.Fatal(err)
	}
	if store.read == nil || store.writer == nil {
		t.Fatal("store did not initialize read pool and writer")
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteLoop_submitRunsOnDedicatedConnection(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir()+"/store.db", slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if err := store.writer.submit(context.Background(), func(ctx context.Context, conn *sql.Conn) error {
		_, err := conn.ExecContext(ctx, `INSERT INTO cache(adapter, input_hash, response) VALUES ('test', 'hash', 'value')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	var got string
	if err := store.read.QueryRow(`SELECT response FROM cache WHERE adapter = 'test'`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != "value" {
		t.Fatalf("response = %q, want value", got)
	}
}

func TestWriteLoop_enqueueHonorsCanceledContext(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir()+"/store.db", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if store.writer.enqueue(ctx, func(context.Context, *sql.Conn) error { return nil }) {
		t.Fatal("canceled enqueue was accepted")
	}
}
