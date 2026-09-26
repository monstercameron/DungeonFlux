package wire

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestSimulatedGame_PhoneSessionReachesEndInFakeMode(t *testing.T) {
	t.Setenv("DF_DEBUG_TOKEN", "sim-debug")
	cfg, err := config.Load(filepath.Join("..", "..", "config", "fake.json"))
	if err != nil {
		t.Fatalf("load fake config: %v", err)
	}
	cfg.Server.Port = 18170
	cfg.Server.DataDir = filepath.Join(t.TempDir(), "runtime")
	cfg.Server.RoomCode, cfg.Server.HostToken = "DF-SIM", "sim-host"
	cfg.Server.DMToken = "sim-dm"
	app, err := Build(context.Background(), cfg, []byte("sim-seed"))
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	defer func() { _ = app.Close() }()
	server := httptest.NewServer(app.Handler())
	defer server.Close()
	conn, err := grpctunnel.Dial(server.URL+"/grpc", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial server: %v", err)
	}
	defer conn.Close()
	session := df.NewSessionServiceClient(conn)
	host := df.NewHostServiceClient(conn)
	seats := joinSimulationSeats(t, session, cfg.Server.RoomCode)
	watchCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	watch, err := session.Watch(watchCtx, &df.WatchRequest{SeatToken: seats[0]})
	if err != nil {
		t.Fatalf("watch seat 1: %v", err)
	}
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_START)
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
	waitSimPhase(t, watch, "exploration")
	phoneAct(t, session, seats[0], "talk_vell", "")
	waitSimPhase(t, watch, "conversation")
	phoneSay(t, session, seats[0], "I persuade Mother Vell.")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "check")
	hostForceD20(t, host, cfg.Server.HostToken, 17)
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "resolution")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "exploration")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "hook_event")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "combat")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "cliffhanger")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "end")
	logs, err := os.ReadFile(filepath.Join(cfg.Server.DataDir, "logs", "server.jsonl"))
	if err != nil {
		t.Fatalf("read effect log: %v", err)
	}
	if count := strings.Count(string(logs), `"msg":"effect executed"`); count < 5 {
		t.Fatalf("effect log records = %d, want at least 5", count)
	}
}

func joinSimulationSeats(t *testing.T, session df.SessionServiceClient, room string) []string {
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

func phoneAct(t *testing.T, session df.SessionServiceClient, token, move, arg string) {
	t.Helper()
	response, err := session.Act(context.Background(), &df.ActRequest{SeatToken: token, MoveId: move, Arg: arg})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("phone act %q: response=%v error=%v", move, response, err)
	}
}

func phoneSay(t *testing.T, session df.SessionServiceClient, token, text string) {
	t.Helper()
	response, err := session.Say(context.Background(), &df.SayRequest{SeatToken: token, Text: text})
	if err != nil || !response.GetAccepted() {
		t.Fatalf("phone say: response=%v error=%v", response, err)
	}
}

func hostCommand(t *testing.T, host df.HostServiceClient, token string, command df.HostCommandKind) {
	t.Helper()
	response, err := host.Command(context.Background(), &df.HostCommand{HostToken: token, Command: command})
	if err != nil || !response.GetOk() {
		t.Fatalf("host command %v: response=%v error=%v", command, response, err)
	}
}

func hostForceD20(t *testing.T, host df.HostServiceClient, token string, d20 int32) {
	t.Helper()
	response, err := host.Command(context.Background(), &df.HostCommand{HostToken: token, Command: df.HostCommandKind_HOST_COMMAND_KIND_FORCE_D20, D20: d20})
	if err != nil || !response.GetOk() {
		t.Fatalf("host force d20: response=%v error=%v", response, err)
	}
}

func waitSimPhase(t *testing.T, watch df.SessionService_WatchClient, want string) {
	t.Helper()
	for {
		message, err := watch.Recv()
		if err != nil {
			t.Fatalf("waiting for phase %q: %v", want, err)
		}
		state := message.GetState()
		if state != nil && state.GetPhase() == want {
			return
		}
	}
}
