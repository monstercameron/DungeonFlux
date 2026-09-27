package wire

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	_ "modernc.org/sqlite"
)

func TestBuild_EventsPersistPerRunAcrossReset(t *testing.T) {
	cfg := testConfig(t)
	app, err := Build(context.Background(), cfg, []byte("rehearsal"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	ctx := context.Background()
	ack := make(chan domain.Ack, 1)
	if !app.room.Post(ctx, domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}, Reply: ack}) {
		t.Fatal("reset was not accepted by room")
	}
	if response := <-ack; !response.Accepted {
		t.Fatalf("reset ack = %+v", response)
	}
	if err := app.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(cfg.Server.DataDir, "dungeonflux.db"))+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var runs, runsWithEvents, orphans int
	if err := db.QueryRow(`SELECT count(*) FROM runs`).Scan(&runs); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(DISTINCT run_id) FROM events WHERE run_id IN (SELECT id FROM runs)`).Scan(&runsWithEvents); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM events WHERE run_id NOT IN (SELECT id FROM runs)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if logs, _ := os.ReadFile(filepath.Join(cfg.Server.DataDir, "logs", "server.jsonl")); runs != 2 {
		t.Logf("server log:\n%s", logs)
	}
	if runs != 2 || runsWithEvents < 1 || orphans != 0 {
		t.Fatalf("runs = %d, runs with events = %d, orphan events = %d; want 2 runs, the first logging the reset, no orphans", runs, runsWithEvents, orphans)
	}
}
