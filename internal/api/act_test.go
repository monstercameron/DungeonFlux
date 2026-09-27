package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func TestSessionServer_ActAndSayPostEvents(t *testing.T) {
	inbox := &fakes.FakeInbox{PostResult: true}
	server, err := NewSessionServer(inbox, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	act, err := server.Act(context.Background(), &df.ActRequest{SeatToken: joined.GetSeatToken(), MoveId: "move", Arg: "north", TargetId: "door", Cell: &df.Cell{C: 2, R: 3}})
	if err != nil || !act.GetAccepted() {
		t.Fatalf("Act() = %+v, %v", act, err)
	}
	say, err := server.Say(context.Background(), &df.SayRequest{SeatToken: joined.GetSeatToken(), Text: "hello there"})
	if err != nil || !say.GetAccepted() || say.GetUtteranceId() == "" {
		t.Fatalf("Say() = %+v, %v", say, err)
	}
	if len(inbox.Calls) != 3 {
		t.Fatalf("posted calls = %d, want join, act, say", len(inbox.Calls))
	}
	if event, ok := inbox.Calls[1].Envelope.Event.(domain.Act); !ok || event.Cell != (domain.Cell{C: 2, R: 3}) {
		t.Fatalf("act event = %#v", inbox.Calls[1].Envelope.Event)
	}
	if event, ok := inbox.Calls[2].Envelope.Event.(domain.Say); !ok || event.Text != "hello there" {
		t.Fatalf("say event = %#v", inbox.Calls[2].Envelope.Event)
	}
}

func TestSessionServer_ActAndSayRejectInvalidRequests(t *testing.T) {
	server, err := NewSessionServer(&fakes.FakeInbox{PostResult: true}, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		act  *df.ActRequest
		say  *df.SayRequest
	}{
		{"missing token", &df.ActRequest{MoveId: "ready"}, nil},
		{"missing move", &df.ActRequest{SeatToken: "bad"}, nil},
		{"missing say token", nil, &df.SayRequest{Text: "hi"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.act != nil {
				if _, err := server.Act(context.Background(), test.act); err == nil {
					t.Fatal("Act() error = nil")
				}
			} else if _, err := server.Say(context.Background(), test.say); err == nil {
				t.Fatal("Say() error = nil")
			}
		})
	}
	joined, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := server.Say(context.Background(), &df.SayRequest{SeatToken: joined.GetSeatToken(), Text: string(make([]rune, 281))}); err == nil {
		t.Fatal("long Say() error = nil")
	}
}

func TestSessionServer_EngineAckRejectsIllegalMove(t *testing.T) {
	engine := &fakes.FakeEngine{Steps: []domain.StepOut{{Ack: &domain.Ack{Reason: "move is illegal"}}}}
	server, err := NewSessionServerWithEngine(&fakes.FakeInbox{PostResult: true}, engine, "ROOM", "HOST", "DM")
	if err != nil {
		t.Fatal(err)
	}
	joined, err := server.Join(context.Background(), &df.JoinRequest{RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE})
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Act(context.Background(), &df.ActRequest{SeatToken: joined.GetSeatToken(), MoveId: "persuade"})
	if err != nil || response.GetAccepted() || response.GetReason() != "move is illegal" {
		t.Fatalf("response = %+v, err = %v", response, err)
	}
}
