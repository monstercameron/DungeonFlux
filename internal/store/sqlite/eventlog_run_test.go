package sqlite

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestAppend_runForeignKey(t *testing.T) {
	tests := []struct {
		name      string
		run       domain.RunID
		wantRows  int
		wantError bool
	}{
		{name: "records for a started run are stored", run: "run-started", wantRows: 1},
		{name: "records without a started run are rejected and logged", run: "", wantRows: 0, wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var logs bytes.Buffer
			path := t.TempDir() + "/store.db"
			store, err := Open(context.Background(), path, slog.New(slog.NewTextHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			if err := store.Start(context.Background(), domain.Run{ID: "run-started", Room: "DF-TEST", Mode: "demo", Seed: []byte("seed")}); err != nil {
				t.Fatal(err)
			}
			record := domain.LogRecord{Seq: 1, Run: tc.run, Kind: domain.Join{Seat: 1}.Kind(), Event: domain.Join{Seat: 1}}
			if err := store.Append(context.Background(), []domain.LogRecord{record}); err != nil {
				t.Fatal(err)
			}
			if err := store.Close(); err != nil {
				t.Fatal(err)
			}
			reopened, err := Open(context.Background(), path, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer reopened.Close()
			var rows int
			if err := reopened.read.QueryRow(`SELECT count(*) FROM events`).Scan(&rows); err != nil {
				t.Fatal(err)
			}
			if rows != tc.wantRows {
				t.Fatalf("event rows = %d, want %d", rows, tc.wantRows)
			}
			if got := strings.Contains(logs.String(), "sqlite write failed"); got != tc.wantError {
				t.Fatalf("write failure logged = %v, want %v; logs:\n%s", got, tc.wantError, logs.String())
			}
		})
	}
}
