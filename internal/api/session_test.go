package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"google.golang.org/grpc/status"
)

func TestSessionServer_JoinPhoneAllocatesAndRestoresSeat(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	first, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	if first.GetSeatId() != "1" || first.GetPlayerNumber() != 1 || first.GetSeatToken() == "" {
		t.Fatalf("join response = %+v", first)
	}
	restored, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE, SeatToken: first.GetSeatToken()})
	if err != nil {
		t.Fatal(err)
	}
	if restored.String() != first.String() {
		t.Fatalf("restored = %v, first = %v", restored, first)
	}
	if len(inbox.Calls) != 1 {
		t.Fatalf("join events = %d, want one for initial join", len(inbox.Calls))
	}
	if event, ok := inbox.Calls[0].Envelope.Event.(domain.Join); !ok || event.Seat != 1 {
		t.Fatalf("join event = %#v", inbox.Calls[0].Envelope.Event)
	}
}

func TestSessionServer_JoinRejectsBadCredentialsAndFullRoom(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		request *df.JoinRequest
		code    int
	}{
		{"nil", nil, 3},
		{"room", &df.JoinRequest{RoomCode: "NOPE", Kind: df.ClientKind_CLIENT_KIND_PHONE}, 7},
		{"kind", &df.JoinRequest{RoomCode: "ROOM"}, 3},
		{"dm", &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_DM}, 7},
		{"host", &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_HOST}, 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := server.Join(context.Background(), test.request)
			if err == nil || int(statusCode(err)) != test.code {
				t.Fatalf("error = %v, want code %d", err, test.code)
			}
		})
	}
	for i := 0; i < 2; i++ {
		if _, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE}); err != nil {
			t.Fatal(err)
		}
	}
	_, err = server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if statusCode(err) != 8 {
		t.Fatalf("full room error = %v, want resource exhausted", err)
	}
}

func TestSessionServer_JoinAllowsDMAndHostTokens(t *testing.T) {
	server, err := NewSessionServer(&fakes.FakeInbox{PostResult: true}, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range []*df.JoinRequest{
		{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_DM, DmToken: "DM"},
		{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_HOST, HostToken: "HOST"},
	} {
		if _, err := server.Join(context.Background(), request); err != nil {
			t.Fatalf("Join(%v) error = %v", request.GetKind(), err)
		}
	}
}

func TestSessionServer_TokenOnlyJoinReattachesPhoneAndKeepsLocale(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	first, err := server.Join(context.Background(), &df.JoinRequest{
		RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE, Locale: "es-MX",
	})
	if err != nil {
		t.Fatal(err)
	}
	reconnected, err := server.Join(context.Background(), &df.JoinRequest{
		RoomCode: "ROOM", SeatToken: first.GetSeatToken(),
	})
	if err != nil {
		t.Fatalf("token-only reattach: %v", err)
	}
	if reconnected.GetSeatId() != first.GetSeatId() || reconnected.GetPlayerNumber() != first.GetPlayerNumber() {
		t.Fatalf("reattached seat = %+v, initial = %+v", reconnected, first)
	}
	if reconnected.GetLocale() != "es" {
		t.Fatalf("reattached locale = %q, want es", reconnected.GetLocale())
	}
	if len(inbox.Calls) != 1 {
		t.Fatalf("reattach posted %d join events, want one", len(inbox.Calls))
	}
}

func TestSessionServer_TokenOnlyJoinRecognizesDMAndHost(t *testing.T) {
	server, err := NewSessionServer(&fakes.FakeInbox{PostResult: true}, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"DM", "HOST"} {
		t.Run(token, func(t *testing.T) {
			request := &df.JoinRequest{RoomCode: "ROOM", SeatToken: token}
			if token == "DM" {
				request.DmToken = token
			} else {
				request.HostToken = token
			}
			if _, err := server.Join(context.Background(), request); err != nil {
				t.Fatalf("token-only %s join: %v", token, err)
			}
		})
	}
}

func statusCode(err error) int {
	if err == nil {
		return 0
	}
	return int(status.Code(err))
}
