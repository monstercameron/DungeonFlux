package wire

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestE2E_ResetRetainsBothPlayerStreamsNamesAndLocales(t *testing.T) {
	app, cfg := buildPathApp(t)
	defer app.Close()
	server := newPathHTTPServer(t, app)
	defer server.Close()
	conn := dialPathGRPC(t, server.URL)
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	session, host := df.NewSessionServiceClient(conn), df.NewHostServiceClient(conn)
	joins := make([]*df.JoinResponse, 0, 2)
	watches := make([]df.SessionService_WatchClient, 0, 2)
	for _, entry := range []struct{ name, locale string }{{"Lyra", "es"}, {"Brom", "en"}} {
		joined, err := session.Join(ctx, &df.JoinRequest{RoomCode: cfg.Server.RoomCode, Kind: df.ClientKind_CLIENT_KIND_PHONE, PlayerName: entry.name, Locale: entry.locale})
		if err != nil {
			t.Fatal(err)
		}
		joins = append(joins, joined)
		watch, err := session.Watch(ctx, &df.WatchRequest{SeatToken: joined.GetSeatToken()})
		if err != nil {
			t.Fatal(err)
		}
		watches = append(watches, watch)
	}
	for range 2 {
		for _, joined := range joins {
			phoneAct(t, session, joined.GetSeatToken(), "ready", "")
		}
		versions := make([]uint64, len(watches))
		for index, watch := range watches {
			versions[index] = receiveLobbySeats(t, watch, 0, true).GetVersion()
		}
		response, err := host.Command(ctx, &df.HostCommand{HostToken: cfg.Server.HostToken, Command: df.HostCommandKind_HOST_COMMAND_KIND_RESET})
		if err != nil || !response.GetOk() {
			t.Fatalf("reset = %v, %v", response, err)
		}
		for index, watch := range watches {
			restored := receiveLobbySeats(t, watch, versions[index], false)
			if restored.GetPhase() != "lobby" || restored.GetPhone().GetLocale() != joins[index].GetLocale() {
				t.Fatalf("reset phone = %v", restored)
			}
		}
	}
	// Original seat tokens must still perform actions after repeated resets.
	for _, joined := range joins {
		phoneAct(t, session, joined.GetSeatToken(), "ready", "")
	}
	for _, watch := range watches {
		receiveLobbySeats(t, watch, 0, true)
	}
}

func receiveLobbySeats(t *testing.T, watch df.SessionService_WatchClient, after uint64, ready bool) *df.ScreenState {
	t.Helper()
	for {
		message, err := watch.Recv()
		if err != nil {
			t.Fatalf("original stream stopped: %v", err)
		}
		state := message.GetState()
		if state.GetVersion() <= after {
			continue
		}
		seats := state.GetPhone().GetSeats()
		if len(seats) != 2 || !seats[0].GetJoined() || !seats[1].GetJoined() || seats[0].GetReady() != ready || seats[1].GetReady() != ready {
			continue
		}
		if seats[0].GetName() != "Lyra" || seats[0].GetLocale() != "es" || seats[1].GetName() != "Brom" || seats[1].GetLocale() != "en" {
			t.Fatalf("reset lost identity: %v", seats)
		}
		return state
	}
}
