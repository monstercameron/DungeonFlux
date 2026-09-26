package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func TestSessionServer_AudioClientForTokenResolvesDMAndPhone(t *testing.T) {
	server, err := NewSessionServer(&fakes.FakeInbox{PostResult: true}, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name  string
		token string
		kind  df.ClientKind
		seat  int
		ok    bool
	}{
		{name: "dm", token: "DM", kind: df.ClientKind_CLIENT_KIND_DM, ok: true},
		{name: "phone", token: joined.SeatToken, kind: df.ClientKind_CLIENT_KIND_PHONE, seat: 1, ok: true},
		{name: "host excluded", token: "HOST", ok: false},
		{name: "unknown excluded", token: "missing", ok: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, ok := server.AudioClientForToken(test.token)
			if ok != test.ok || (ok && (client.Kind != test.kind || client.PlayerNumber != test.seat)) {
				t.Fatalf("client=%+v ok=%v", client, ok)
			}
		})
	}
}
