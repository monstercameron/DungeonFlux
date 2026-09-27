package wire

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"path/filepath"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "modernc.org/sqlite"
)

// A phone that resubscribes (after an idle or dropped stream) must not post a
// second join: each join steps the engine and redraws every screen. The 2026-09-27
// playtest logged one join per phone every 25 s from the idle resubscribe.
func TestWatch_resubscribeDoesNotRepostJoin(t *testing.T) {
	cfg := testConfig(t)
	cfg.Server.RoomCode = "DF-WATCH"
	app, err := Build(context.Background(), cfg, []byte("watch-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	server := httptest.NewServer(app.Handler())
	conn, err := grpctunnel.Dial(server.URL+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("tunnel dial: %v", err)
	}
	client := df.NewSessionServiceClient(conn)
	joined, err := client.Join(context.Background(), &df.JoinRequest{RoomCode: cfg.Server.RoomCode, Kind: df.ClientKind_CLIENT_KIND_PHONE, PlayerName: "Ana"})
	if err != nil {
		t.Fatalf("join: %v", err)
	}
	for i := range 3 {
		ctx, cancel := context.WithCancel(context.Background())
		stream, err := client.Watch(ctx, &df.WatchRequest{SeatToken: joined.GetSeatToken()})
		if err != nil {
			t.Fatalf("watch %d: %v", i, err)
		}
		if message, err := stream.Recv(); err != nil || message.GetState() == nil {
			t.Fatalf("watch %d first message = %v, %v", i, message, err)
		}
		cancel()
	}
	_ = conn.Close()
	server.Close()
	if err := app.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(cfg.Server.DataDir, "dungeonflux.db"))+"?mode=ro")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var joins int
	if err := db.QueryRow(`SELECT count(*) FROM events WHERE kind = 'join'`).Scan(&joins); err != nil {
		t.Fatal(err)
	}
	if joins != 1 {
		t.Fatalf("join events = %d after one Join and three Watch subscriptions, want 1", joins)
	}
}
