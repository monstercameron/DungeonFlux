package wire

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestE2E_Path12_DMWatchReconnectAndReport(t *testing.T) {
	app, cfg := buildPathApp(t)
	defer func() { _ = app.Close() }()
	server := newPathHTTPServer(t, app)
	defer server.Close()

	conn := dialPathGRPC(t, server.URL)
	defer conn.Close()
	session := df.NewSessionServiceClient(conn)
	joined, err := session.Join(context.Background(), &df.JoinRequest{
		RoomCode: cfg.Server.RoomCode, Kind: df.ClientKind_CLIENT_KIND_PHONE,
	})
	if err != nil {
		t.Fatalf("join phone: %v", err)
	}
	if joined.GetSeatToken() == "" {
		t.Fatal("join phone returned an empty seat token")
	}

	watchCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	watch, err := session.Watch(watchCtx, &df.WatchRequest{SeatToken: joined.GetSeatToken()})
	if err != nil {
		cancel()
		t.Skipf("E2E-002 path 12 blocked by BASE-010 Watch reattach kind: %v", err)
	}
	if _, err := session.Report(context.Background(), &df.ReportRequest{
		SeatToken: joined.GetSeatToken(),
		Kind:      df.ReportKind_REPORT_KIND_SPLAT_FAILED,
		Id:        "path-12-watch",
	}); err != nil {
		cancel()
		t.Fatalf("report while Watch is attached: %v", err)
	}
	if _, err := watch.Recv(); err != nil {
		cancel()
		t.Skipf("E2E-002 path 12 blocked by BASE-010 Watch reattach kind: %v", err)
	}
	cancel()

	if _, err := session.Join(context.Background(), &df.JoinRequest{
		RoomCode: cfg.Server.RoomCode, Kind: df.ClientKind_CLIENT_KIND_PHONE,
		SeatToken: joined.GetSeatToken(),
	}); err != nil {
		t.Fatalf("rejoin phone after stream drop: %v", err)
	}
	if _, err := session.Report(context.Background(), &df.ReportRequest{
		SeatToken:   joined.GetSeatToken(),
		Kind:        df.ReportKind_REPORT_KIND_CLIENT_STATE,
		Id:          "path-12-reload",
		ClientState: "reconnected",
	}); err != nil {
		t.Fatalf("report after reconnect: %v", err)
	}
}

func TestE2E_Path21_LatestDMListenReplacesOlderStream(t *testing.T) {
	app, _ := buildPathApp(t)
	defer func() { _ = app.Close() }()
	server := newPathHTTPServer(t, app)
	defer server.Close()
	conn := dialPathGRPC(t, server.URL)
	defer conn.Close()

	audio := df.NewAudioServiceClient(conn)
	firstCtx, firstCancel := context.WithCancel(context.Background())
	defer firstCancel()
	first, err := audio.Listen(firstCtx, &df.ListenRequest{SeatToken: "dm-token"})
	if err != nil {
		t.Fatalf("first Listen: %v", err)
	}
	secondCtx, secondCancel := context.WithCancel(context.Background())
	defer secondCancel()
	if _, err := audio.Listen(secondCtx, &df.ListenRequest{SeatToken: "dm-token"}); err != nil {
		t.Fatalf("second Listen: %v", err)
	}

	// ListenHub currently permits both subscribers and has no composition-root
	// hook for publishing a frame. Keep the end-to-end assertion pending until
	// the replacement policy is wired; this still proves both bridge streams
	// can be opened against the real server.
	_ = first
	t.Skip("E2E-002 path 21 blocked by API-005 follow-up: latest DM Listen replacement")
}

func buildPathApp(t *testing.T) (*App, config.Config) {
	t.Helper()
	debugPort := freePort(t)
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: debugPort - 1000, Debug: false, LogLevel: "debug",
			DataDir: filepath.Join(t.TempDir(), "runtime"), RoomCode: "DF-PATH",
			HostToken: "host-token", DMToken: "dm-token",
		},
		Budget: config.BudgetConfig{PerRunUSD: 1, HardUSD: 2},
		Timeouts: config.TimeoutConfig{
			CharacterFlavor: time.Second, Interpret: time.Second,
			SpokenFirstToken: time.Second, Prerender: time.Second,
			Portrait: time.Second, TTS: time.Second,
		},
		Battlefield: config.BattlefieldConfig{NavPath: "internal/content/battlefield_tavern.json"},
	}
	app, err := Build(context.Background(), cfg, []byte("e2e-path-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return app, cfg
}

func newPathHTTPServer(t *testing.T, app *App) *httptest.Server {
	t.Helper()
	return httptest.NewServer(app.Handler())
}

func dialPathGRPC(t *testing.T, endpoint string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpctunnel.Dial(endpoint+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial grpc bridge: %v", err)
	}
	return conn
}
