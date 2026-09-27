package wire

import (
	"context"
	"strings"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestE2E_HostStartWaitsForPhoneReadiness(t *testing.T) {
	app, cfg := buildPathApp(t)
	defer app.Close()
	server := newPathHTTPServer(t, app)
	defer server.Close()
	conn := dialPathGRPC(t, server.URL)
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, host := df.NewSessionServiceClient(conn), df.NewHostServiceClient(conn)
	start := func(accepted bool, reason string) {
		t.Helper()
		ack, err := host.Command(ctx, &df.HostCommand{HostToken: cfg.Server.HostToken, Command: df.HostCommandKind_HOST_COMMAND_KIND_START})
		if err != nil || ack.GetOk() != accepted || !strings.Contains(ack.GetReason(), reason) {
			t.Fatalf("start accepted=%v, reason=%q: ack=%v error=%v", accepted, reason, ack, err)
		}
	}
	start(false, "join")
	tokens := joinSimulationSeats(t, session, cfg.Server.RoomCode)
	watch, err := session.Watch(ctx, &df.WatchRequest{SeatToken: tokens[0]})
	if err != nil {
		t.Fatal(err)
	}
	start(false, "Ready")
	phoneAct(t, session, tokens[0], "ready", "")
	start(false, "Ready")
	phoneAct(t, session, tokens[1], "ready", "")
	start(true, "")
	waitSimPhase(t, watch, "creation")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_RESET)
	waitSimPhase(t, watch, "lobby")
	start(false, "Ready")
	hostCommand(t, host, cfg.Server.HostToken, df.HostCommandKind_HOST_COMMAND_KIND_SKIP)
	waitSimPhase(t, watch, "creation")
}
