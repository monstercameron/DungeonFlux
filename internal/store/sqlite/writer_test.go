package sqlite

import (
	"context"
	"database/sql"
	"log/slog"
	"sync"
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

func TestWriteLoop_writesAfterCloseFailCleanly(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir()+"/store.db", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	noop := func(context.Context, *sql.Conn) error { return nil }
	if err := store.writer.submit(context.Background(), noop); err == nil {
		t.Fatal("submit after close succeeded, want an error instead of a panic")
	}
	if store.writer.enqueue(context.Background(), noop) {
		t.Fatal("enqueue after close accepted a write")
	}
}

func TestWriteLoop_closeRacesConcurrentWrites(t *testing.T) {
	store, err := Open(context.Background(), t.TempDir()+"/store.db", nil)
	if err != nil {
		t.Fatal(err)
	}
	noop := func(context.Context, *sql.Conn) error { return nil }
	var writers sync.WaitGroup
	for i := 0; i < 8; i++ {
		writers.Add(1)
		go func() {
			defer writers.Done()
			for j := 0; j < 50; j++ {
				_ = store.writer.submit(context.Background(), noop)
				store.writer.enqueue(context.Background(), noop)
			}
		}()
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	writers.Wait()
}
