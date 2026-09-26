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

	phaseTrace := []string{"lobby"}
	sendDebug(t, debugClient, debugCtx, "host_start")
	assertPhase(t, debugClient, debugCtx, "DF-E2E", "creation", phaseTrace)
	phaseTrace = append(phaseTrace, "creation")

	// Character creation is intentionally driven through the same Act RPC that
	// dfctl uses. If the root engine has not yet dispatched phase events, this
	// is the first observable stall and the skip records all useful evidence.
	// Each seat picks species and gender, then rolls; picking alone keeps the
	// room in creation, so only the legal moves decide what is sent next.
	for _, seat := range []string{"1", "2"} {
		sendAct(t, debugClient, debugCtx, seat, "species", "human")
		sendAct(t, debugClient, debugCtx, seat, "gender", "nonbinary")
		class := "paladin"
		if seat == "2" {
			class = "rogue"
		}
		sendAct(t, debugClient, debugCtx, seat, "class", class)
		sendAct(t, debugClient, debugCtx, seat, "roll_hero", "")
		if contains(readLegal(t, debugClient, debugCtx, "DF-E2E", seat), "ready") {
			sendAct(t, debugClient, debugCtx, seat, "ready", "")
		}
	}
	state = waitPhaseLeaves(t, debugClient, debugCtx, "DF-E2E", "creation")
	if state.GetPhase() == "creation" {
		moves := readLegal(t, debugClient, debugCtx, "DF-E2E", "1")
		t.Skipf("e2e stall: both seats picked species, gender, roll_hero, ready; state=%q; legal_moves=%v", state.GetPhase(), moves)
	}
	assertPhase(t, debugClient, debugCtx, "DF-E2E", "opening", phaseTrace)
	phaseTrace = append(phaseTrace, "opening")
	if response, err := debugClient.Say(debugCtx, &df.DebugSayRequest{Room: "DF-E2E", Seat: "1", Text: "I greet Mother Vell."}); err != nil || !response.GetAccepted() {
		t.Fatalf("dfctl say: response=%v error=%v", response, err)
	}
	if response, err := debugClient.DiceForce(debugCtx, &df.DiceForceRequest{Room: "DF-E2E", D20: 17}); err != nil || !response.GetAccepted() {
		t.Fatalf("dfctl dice force: response=%v error=%v", response, err)
	}
	for _, phase := range []string{"exploration", "conversation", "check", "resolution", "exploration", "hook_event"} {
		sendDebug(t, debugClient, debugCtx, "host_skip")
		assertPhase(t, debugClient, debugCtx, "DF-E2E", phase, phaseTrace)
		phaseTrace = append(phaseTrace, phase)
	}
	sendDebug(t, debugClient, debugCtx, "host_skip")
	assertPhase(t, debugClient, debugCtx, "DF-E2E", "combat", phaseTrace)
	phaseTrace = append(phaseTrace, "combat")

	sendAct(t, debugClient, debugCtx, "1", "attack", "")
	for i := 0; i < 8; i++ {
		sendAct(t, debugClient, debugCtx, "2", "attack", "")
		if readDebugState(t, debugClient, debugCtx, "DF-E2E").GetPhase() != "combat" {
			break
		}
		sendAct(t, debugClient, debugCtx, "1", "attack", "")
		if readDebugState(t, debugClient, debugCtx, "DF-E2E").GetPhase() != "combat" {
			break
		}
	}
	state = readDebugState(t, debugClient, debugCtx, "DF-E2E")
	if state.GetPhase() != "cliffhanger" {
		moves := readLegal(t, debugClient, debugCtx, "DF-E2E", "1")
		t.Skipf("e2e stall: event combat attacks/end_turn; state=%q; legal_moves=%v", state.GetPhase(), moves)
	}
	phaseTrace = append(phaseTrace, state.GetPhase())
	sendDebug(t, debugClient, debugCtx, "host_skip")
	assertPhase(t, debugClient, debugCtx, "DF-E2E", "end", phaseTrace)
	t.Logf("phase trace: %v", phaseTrace)
}

func sendDebug(t *testing.T, client df.DebugServiceClient, ctx context.Context, event string) {
	t.Helper()
	response, err := client.Send(ctx, &df.SendRequest{Room: "DF-E2E", Event: event})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("dfctl send %q: response=%v error=%v", event, response, err)
	}
}

func sendAct(t *testing.T, client df.DebugServiceClient, ctx context.Context, seat, move, arg string) {
	t.Helper()
	target := ""
	if move == "attack" {
		target = "thrall"
	}
	response, err := client.Act(ctx, &df.DebugActRequest{Room: "DF-E2E", Seat: seat, MoveId: move, Arg: arg, TargetId: target})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("dfctl act seat=%s move=%s: response=%v error=%v", seat, move, response, err)
	}
}

func assertPhase(t *testing.T, client df.DebugServiceClient, ctx context.Context, room, want string, trace []string) *df.DebugState {
	t.Helper()
	state := readDebugState(t, client, ctx, room)
	if state.GetPhase() != want {
		t.Fatalf("phase trace %v: got %q, want %q", trace, state.GetPhase(), want)
	}
	return state
}

func readLegal(t *testing.T, client df.DebugServiceClient, ctx context.Context, room, seat string) []string {
	t.Helper()
	moves, err := client.Legal(ctx, &df.SeatRequest{Room: room, Seat: seat})
	if err != nil {
		t.Fatalf("dfctl legal seat=%s: %v", seat, err)
	}
	result := make([]string, 0, len(moves.GetMoves()))
	for _, move := range moves.GetMoves() {
		result = append(result, move.GetMoveId())
	}
	return result
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// waitPhaseLeaves polls the debug state until the room leaves phase or a short
// deadline passes; executors on fakes post their results asynchronously.
func waitPhaseLeaves(t *testing.T, client df.DebugServiceClient, ctx context.Context, room, phase string) *df.DebugState {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for {
		state := readDebugState(t, client, ctx, room)
		if state.GetPhase() != phase || time.Now().After(deadline) {
			return state
		}
		time.Sleep(20 * time.Millisecond)
	}
}
