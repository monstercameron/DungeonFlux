package wire

import (
	"context"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestE2E_HookLeaveReachesCombatWithoutHostSkip(t *testing.T) {
	for _, timersEnabled := range []bool{true, false} {
		t.Run(timerCaseName(timersEnabled), func(t *testing.T) {
			runHookPath(t, timersEnabled)
		})
	}
}

func runHookPath(t *testing.T, timersEnabled bool) {
	t.Helper()
	debugPort := freePort(t)
	t.Setenv("DF_DEBUG_TOKEN", "eng29-e2e-token")
	cfg := config.Config{
		Server: config.ServerConfig{
			Port: debugPort - 1000, Debug: true, LogLevel: "debug",
			DataDir: filepath.Join(t.TempDir(), "runtime"), RoomCode: "DF-HOOK",
			HostToken: "hook-host", DMToken: "hook-dm",
		},
		Features: config.FeatureConfig{TurnTimers: timersEnabled},
		Budget:   config.BudgetConfig{PerRunUSD: 1, HardUSD: 2},
		Timeouts: config.TimeoutConfig{
			CharacterFlavor: time.Second, Interpret: time.Second,
			SpokenFirstToken: time.Second, Prerender: time.Second,
			Portrait: time.Second, TTS: time.Second,
		},
		Battlefield: config.BattlefieldConfig{NavPath: "internal/content/battlefield_tavern.json"},
	}
	app, err := Build(context.Background(), cfg, []byte("eng29-hook-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = app.Close() }()
	server := httptest.NewServer(app.Handler())
	defer server.Close()

	conn, err := grpctunnel.Dial(server.URL+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial app: %v", err)
	}
	defer conn.Close()
	session := df.NewSessionServiceClient(conn)
	seats := joinHookSeats(t, session, cfg.Server.RoomCode)
	watchCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	watch, err := session.Watch(watchCtx, &df.WatchRequest{SeatToken: seats[0]})
	if err != nil {
		t.Fatalf("watch seat 1: %v", err)
	}

	debugConn, err := grpc.NewClient("127.0.0.1:"+strconv.Itoa(debugPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial debug: %v", err)
	}
	defer debugConn.Close()
	debugClient := df.NewDebugServiceClient(debugConn)
	debugCtx := metadata.AppendToOutgoingContext(context.Background(), "x-df-debug-token", "eng29-e2e-token")

	sendHookDebug(t, debugClient, debugCtx, "host_start", "")
	waitSimPhase(t, watch, "creation")
	for seat, token := range seats {
		phoneAct(t, session, token, "species", "human")
		phoneAct(t, session, token, "gender", "nonbinary")
		class := "paladin"
		if seat == 1 {
			class = "rogue"
		}
		phoneAct(t, session, token, "class", class)
		phoneAct(t, session, token, "roll_hero", "")
		phoneAct(t, session, token, "ready", "")
	}
	waitSimPhase(t, watch, "opening")
	sendHookDebug(t, debugClient, debugCtx, "host_skip", "")
	waitSimPhase(t, watch, "exploration")
	phoneAct(t, session, seats[0], "leave", "")
	waitSimPhase(t, watch, "hook_event")
	waitSimPhase(t, watch, "combat")
}

func joinHookSeats(t *testing.T, session df.SessionServiceClient, room string) []string {
	t.Helper()
	tokens := make([]string, 0, 2)
	for seat := 1; seat <= 2; seat++ {
		joined, err := session.Join(context.Background(), &df.JoinRequest{RoomCode: room, Kind: df.ClientKind_CLIENT_KIND_PHONE})
		if err != nil || joined.GetPlayerNumber() != int32(seat) || joined.GetSeatToken() == "" {
			t.Fatalf("join seat %d: response=%v error=%v", seat, joined, err)
		}
		tokens = append(tokens, joined.GetSeatToken())
	}
	return tokens
}

func sendHookDebug(t *testing.T, client df.DebugServiceClient, ctx context.Context, event, seat string) {
	t.Helper()
	if event == "host_start" || event == "host_skip" {
		response, err := client.Send(ctx, &df.SendRequest{Room: "DF-HOOK", Event: event})
		if err != nil || !response.GetAccepted() {
			t.Fatalf("send %q: response=%v error=%v", event, response, err)
		}
		return
	}
	response, err := client.Act(ctx, &df.DebugActRequest{Room: "DF-HOOK", Seat: seat, MoveId: event})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("act %q: response=%v error=%v", event, response, err)
	}
}

func timerCaseName(enabled bool) string {
	if enabled {
		return "timers_on"
	}
	return "timers_off"
}
