package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func TestSessionServer_WatchSeat(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE, PlayerName: "Ana"})
	if err != nil {
		t.Fatal(err)
	}
	posted := len(inbox.Calls)
	tests := []struct {
		name     string
		token    string
		wantSeat domain.SeatID
		wantOK   bool
	}{
		{name: "a seated phone token resolves to its seat", token: joined.GetSeatToken(), wantSeat: 1, wantOK: true},
		{name: "the DM token is not a phone seat", token: "DM"},
		{name: "an unknown token is not a seat", token: "nope"},
		{name: "an empty token is not a seat", token: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			seat, ok := server.WatchSeat(tc.token)
			if seat != tc.wantSeat || ok != tc.wantOK {
				t.Fatalf("WatchSeat(%q) = %d, %v; want %d, %v", tc.token, seat, ok, tc.wantSeat, tc.wantOK)
			}
		})
	}
	if len(inbox.Calls) != posted {
		t.Fatalf("WatchSeat posted %d events, want none", len(inbox.Calls)-posted)
	}
}
