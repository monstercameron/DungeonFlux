package wire

import (
	"context"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
)

func TestOpenStorage_ExposesAllPortsAndPersists(t *testing.T) {
	storage, err := OpenStorage(context.Background(), filepath.Join(t.TempDir(), "runtime"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("OpenStorage() error = %v", err)
	}
	if storage.EventLog == nil || storage.Runs == nil || storage.Assets == nil || storage.Cache == nil || storage.Recordings == nil {
		t.Fatal("OpenStorage() did not expose all storage ports")
	}
	if err := storage.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestOpenStorage_RequiresDataDirectory(t *testing.T) {
	if _, err := OpenStorage(context.Background(), "", nil); err == nil {
		t.Fatal("OpenStorage() accepted empty data directory")
	}
}
