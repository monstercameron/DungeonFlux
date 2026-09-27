package wire

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/store/sqlite"
)

// Storage holds all SQLite-backed ports used by the runtime composition root.
type Storage struct {
	Store      *sqlite.Store
	EventLog   ports.EventLog
	Runs       ports.Runs
	Assets     ports.Assets
	Cache      ports.Cache
	Recordings ports.Recordings
}

// OpenStorage opens the SQLite database below dataDir and exposes every
// persistence port over the same single-writer store.
func OpenStorage(ctx context.Context, dataDir string, logger *slog.Logger) (*Storage, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("wire: storage data directory is required")
	}
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return nil, fmt.Errorf("wire: create storage data directory: %w", err)
	}
	store, err := sqlite.Open(ctx, filepath.Join(dataDir, "dungeonflux.db"), logger)
	if err != nil {
		return nil, fmt.Errorf("wire: open storage: %w", err)
	}
	return &Storage{Store: store, EventLog: store, Runs: store, Assets: sqlite.NewAssets(store), Cache: sqlite.NewCache(store), Recordings: sqlite.NewRecordings(store)}, nil
}

// Close releases the shared SQLite store.
func (s *Storage) Close() error {
	if s == nil || s.Store == nil {
		return nil
	}
	return s.Store.Close()
}
