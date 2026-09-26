package wire

import (
	"context"
	"fmt"
	"net"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestE2E_DfctlRunThroughLobby(t *testing.T) {
	debugPort := freePort(t)
	t.Setenv("DF_DEBUG_TOKEN", "e2e-debug-token")
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: debugPort - 1000, Debug: true, LogLevel: "debug",
			DataDir: filepath.Join(t.TempDir(), "runtime"), RoomCode: "DF-E2E",
			HostToken: "e2e-host", DMToken: "e2e-dm",
		},
		Budget: config.BudgetConfig{PerRunUSD: 1, HardUSD: 2},
		Timeouts: config.TimeoutConfig{
			CharacterFlavor: time.Second, Interpret: time.Second,
			SpokenFirstToken: time.Second, Prerender: time.Second,
			Portrait: time.Second, TTS: time.Second,
		},
		Battlefield: config.BattlefieldConfig{NavPath: "internal/content/battlefield_tavern.json"},
	}
	app, err := Build(context.Background(), cfg, []byte("e2e-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = app.Close() }()
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	debugConn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", debugPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("debug grpc client: %v", err)
	}
	defer debugConn.Close()
	debugClient := df.NewDebugServiceClient(debugConn)
	debugCtx := metadata.AppendToOutgoingContext(context.Background(), "x-df-debug-token", "e2e-debug-token")

	state := readDebugState(t, debugClient, debugCtx, "DF-E2E")
	if state.GetPhase() != "lobby" {
		t.Fatalf("initial phase = %q, want lobby", state.GetPhase())
	}
	joinSeats(t, server.URL, cfg.Server.RoomCode)

	act, err := debugClient.Act(debugCtx, &df.DebugActRequest{Room: "DF-E2E", Seat: "1", MoveId: "ready"})
	if err != nil || !act.GetAccepted() {
		t.Fatalf("dfctl act: response=%v error=%v", act, err)
	}
	say, err := debugClient.Say(debugCtx, &df.DebugSayRequest{Room: "DF-E2E", Seat: "1", Text: "I greet Mother Vell."})
	if err != nil || !say.GetAccepted() {
		t.Fatalf("dfctl say: response=%v error=%v", say, err)
	}
	dice, err := debugClient.DiceForce(debugCtx, &df.DiceForceRequest{Room: "DF-E2E", D20: 10})
	if err != nil || !dice.GetAccepted() {
		t.Fatalf("dfctl dice force: response=%v error=%v", dice, err)
	}
	state = readDebugState(t, debugClient, debugCtx, "DF-E2E")
	if state.GetPhase() != "lobby" {
		t.Fatalf("phase after reachable dfctl commands = %q, want lobby", state.GetPhase())
	}
	t.Logf("reachable phase trace: %s", state.GetPhase())
	t.Skip("phase progression is pending ENG-014, COMBAT-008, BASE-010, and BASE-011")
}

func joinSeats(t *testing.T, endpoint, room string) {
	t.Helper()
	conn, err := grpctunnel.Dial(endpoint+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("session grpc client: %v", err)
	}
	defer conn.Close()
	client := df.NewSessionServiceClient(conn)
	for _, want := range []int32{1, 2} {
		response, err := client.Join(context.Background(), &df.JoinRequest{RoomCode: room, Kind: df.ClientKind_CLIENT_KIND_PHONE})
		if err != nil {
			t.Fatalf("join seat %d: %v", want, err)
		}
		if response.GetPlayerNumber() != want || response.GetSeatToken() == "" {
			t.Fatalf("join seat %d response = %v", want, response)
		}
	}
}

func readDebugState(t *testing.T, client df.DebugServiceClient, ctx context.Context, room string) *df.DebugState {
	t.Helper()
	state, err := client.State(ctx, &df.DebugRoom{Room: room})
	if err != nil {
		t.Fatalf("dfctl state: %v", err)
	}
	return state
}

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	if port <= 1000 {
		t.Fatal("reserved port is too low for debug port offset")
	}
	return port
}
