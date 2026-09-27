package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func testLocaleView() domain.View {
	return domain.View{
		Path:   vocab.StateConversation,
		Locale: "en",
		Seats: []domain.SeatView{
			{
				Seat:       1,
				StatusText: "Mother Vell is speaking",
				Moves: []domain.MoveView{
					{ID: vocab.MoveAttack, Label: "Attack the drowned thrall", Enabled: true},
					{ID: vocab.MoveLeave, Label: "Leave", Enabled: false, Reason: "Target out of range (30 ft)"},
				},
			},
		},
	}
}

func TestProjectLocalized_SpanishPhone(t *testing.T) {
	state := ProjectLocalized(testLocaleView(), df.ClientKind_CLIENT_KIND_PHONE, 1, "es")
	if state.GetLocale() != "es" {
		t.Fatalf("screen locale = %q, want es", state.GetLocale())
	}
	phone := state.GetPhone()
	if phone.GetLocale() != "es" {
		t.Fatalf("phone locale = %q, want es", phone.GetLocale())
	}
	moves := phone.GetMoves()
	if moves[0].GetLabel() != "Atacar al ahogado" {
		t.Fatalf("attack label = %q", moves[0].GetLabel())
	}
	if moves[0].GetLabelKey() != "move.attack" {
		t.Fatalf("attack label key = %q", moves[0].GetLabelKey())
	}
	if moves[1].GetReason() != "Objetivo fuera de alcance (30 pies)" {
		t.Fatalf("range reason = %q", moves[1].GetReason())
	}
	if moves[1].GetReasonMsg().GetKey() != "reason.OUT_OF_RANGE" {
		t.Fatalf("reason key = %q", moves[1].GetReasonMsg().GetKey())
	}
	if moves[1].GetReasonMsg().GetArgs()["ft"] != "30" {
		t.Fatalf("reason args = %v", moves[1].GetReasonMsg().GetArgs())
	}
	if phone.GetStatusText() != "Madre Vell está hablando" {
		t.Fatalf("status = %q", phone.GetStatusText())
	}
}

func TestProjectLocalized_EnglishKeepsEngineStrings(t *testing.T) {
	state := ProjectLocalized(testLocaleView(), df.ClientKind_CLIENT_KIND_PHONE, 1, "en")
	phone := state.GetPhone()
	if phone.GetMoves()[0].GetLabel() != "Attack the drowned thrall" {
		t.Fatalf("english label = %q", phone.GetMoves()[0].GetLabel())
	}
	if phone.GetMoves()[1].GetReason() != "Target out of range (30 ft)" {
		t.Fatalf("english reason = %q", phone.GetMoves()[1].GetReason())
	}
}

func TestSession_JoinSettlesLocale(t *testing.T) {
	inbox := &localeTestInbox{}
	server, err := NewSessionServer(inbox, "ROOM", "host-token", "dm-token")
	if err != nil {
		t.Fatal(err)
	}
	response, err := server.Join(context.Background(), &df.JoinRequest{
		RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE, Locale: "es",
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.GetLocale() != "es" {
		t.Fatalf("join locale = %q, want es", response.GetLocale())
	}
	if len(inbox.events) != 1 {
		t.Fatalf("posted events = %d, want 1", len(inbox.events))
	}
	join, ok := inbox.events[0].Event.(domain.Join)
	if !ok || join.Locale != "es" {
		t.Fatalf("join event = %#v", inbox.events[0].Event)
	}
	unknown, err := server.Join(context.Background(), &df.JoinRequest{
		RoomCode: "ROOM", Kind: df.ClientKind_CLIENT_KIND_PHONE, Locale: "fr",
	})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.GetLocale() != "en" {
		t.Fatalf("unknown locale settled = %q, want en", unknown.GetLocale())
	}
}

func TestWatchHub_SendsSeatLocale(t *testing.T) {
	hub := NewWatchHub()
	hub.RememberLocale(1, "es")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := hub.Subscribe(ctx, df.ClientKind_CLIENT_KIND_PHONE, 1)
	hub.Publish(testLocaleView())
	message := receiveWatch(t, sub.Messages())
	state := message.GetState()
	if state.GetLocale() != "es" {
		t.Fatalf("watch locale = %q, want es", state.GetLocale())
	}
	if state.GetPhone().GetMoves()[0].GetLabel() != "Atacar al ahogado" {
		t.Fatalf("watch label = %q", state.GetPhone().GetMoves()[0].GetLabel())
	}
}

type localeTestInbox struct {
	events []domain.Envelope
}

func (b *localeTestInbox) Post(_ context.Context, env domain.Envelope) bool {
	b.events = append(b.events, env)
	return true
}
