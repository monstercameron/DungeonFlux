package wire

import (
	"context"
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestE2E_DMLobbySeatNames(t *testing.T) {
	app, cfg := buildPathApp(t)
	defer func() { _ = app.Close() }()
	server := newPathHTTPServer(t, app)
	defer server.Close()
	conn := dialPathGRPC(t, server.URL)
	defer conn.Close()

	session := df.NewSessionServiceClient(conn)
	if _, err := session.Join(context.Background(), &df.JoinRequest{
		RoomCode: cfg.Server.RoomCode, Kind: df.ClientKind_CLIENT_KIND_DM, DmToken: cfg.Server.DMToken,
	}); err != nil {
		t.Fatalf("join DM: %v", err)
	}
	watchCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	watch, err := session.Watch(watchCtx, &df.WatchRequest{SeatToken: cfg.Server.DMToken})
	if err != nil {
		t.Fatalf("watch DM: %v", err)
	}
	joinNamedSeats(t, session, cfg.Server.RoomCode)

	want := map[string]string{"1": "Lyra", "2": "Brom"}
	for {
		message, recvErr := watch.Recv()
		if recvErr != nil {
			t.Fatalf("receive DM view: %v", recvErr)
		}
		if dmSeatNamesMatch(message.GetState().GetDm().GetSeats(), want) {
			return
		}
	}
}

func joinNamedSeats(t *testing.T, session df.SessionServiceClient, room string) {
	t.Helper()
	for _, join := range []struct {
		want int32
		name string
	}{{want: 1, name: "Lyra"}, {want: 2, name: "Brom"}} {
		response, err := session.Join(context.Background(), &df.JoinRequest{
			RoomCode: room, Kind: df.ClientKind_CLIENT_KIND_PHONE, PlayerName: join.name,
		})
		if err != nil {
			t.Fatalf("join seat %d: %v", join.want, err)
		}
		if response.GetPlayerNumber() != join.want || response.GetSeatToken() == "" {
			t.Fatalf("join seat %d response = %v", join.want, response)
		}
	}
}

func dmSeatNamesMatch(seats []*df.LobbySeat, want map[string]string) bool {
	if len(seats) != len(want) {
		return false
	}
	for _, seat := range seats {
		if !seat.GetJoined() || seat.GetName() != want[seat.GetSeatId()] {
			return false
		}
	}
	return true
}
