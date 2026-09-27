package api

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestProject_LobbyReadinessReachesAllScreensWithoutACharacter(t *testing.T) {
	view := domain.View{Path: vocab.StateLobby, Seats: []domain.SeatView{
		{Seat: 1, PlayerNumber: 1, PlayerName: "Lyra", Connected: true, LobbyReady: true},
		{Seat: 2, PlayerNumber: 2, PlayerName: "Brom", Connected: true},
	}}
	for _, tc := range []struct {
		name string
		kind df.ClientKind
	}{
		{"table", df.ClientKind_CLIENT_KIND_DM}, {"phone", df.ClientKind_CLIENT_KIND_PHONE}, {"host", df.ClientKind_CLIENT_KIND_HOST},
	} {
		t.Run(tc.name, func(t *testing.T) {
			screen := Project(view, tc.kind, 1)
			var seats []*df.LobbySeat
			switch tc.kind {
			case df.ClientKind_CLIENT_KIND_DM:
				seats = screen.GetDm().GetSeats()
			case df.ClientKind_CLIENT_KIND_PHONE:
				seats = screen.GetPhone().GetSeats()
			case df.ClientKind_CLIENT_KIND_HOST:
				seats = screen.GetHost().GetDm().GetSeats()
			}
			if len(seats) != 2 || !seats[0].GetReady() || seats[1].GetReady() {
				t.Fatalf("readiness = %+v", seats)
			}
		})
	}
}
