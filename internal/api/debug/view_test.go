package debug

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestView_SeatReturnsPhoneProjection(t *testing.T) {
	server := viewServer(t)
	state, err := server.View(context.Background(), &df.ViewRequest{Seat: "1"})
	if err != nil {
		t.Fatal(err)
	}
	phone := state.GetPhone()
	if phone == nil || phone.GetCharacter().GetName() != "Astra" || phone.GetStatusText() != "your turn" {
		t.Fatalf("phone = %#v", phone)
	}
}

func TestView_DMReturnsDMProjection(t *testing.T) {
	server := viewServer(t)
	state, err := server.View(context.Background(), &df.ViewRequest{View: "dm"})
	if err != nil {
		t.Fatal(err)
	}
	if state.GetDm() == nil || state.GetDm().GetCallout() != "look" || len(state.GetDm().GetBuildCards()) != 1 {
		t.Fatalf("dm = %#v", state.GetDm())
	}
}

func TestView_RejectsMalformedSeat(t *testing.T) {
	server := viewServer(t)
	if _, err := server.View(context.Background(), &df.ViewRequest{Seat: "x"}); err == nil {
		t.Fatal("malformed seat accepted")
	}
}

func viewServer(t *testing.T) *Server {
	t.Helper()
	engine := &fakes.FakeEngine{ViewValue: domain.View{
		Path: vocab.StateConversation, Callout: "look",
		Seats: []domain.SeatView{{Seat: 1, Build: &domain.BuildCard{Name: "Astra", Class: "Rogue", PlayerNumber: 1}, Character: &domain.Character{Name: "Astra"}, StatusText: "your turn"}},
	}}
	server, err := NewServer(engine, &fakes.FakeInbox{PostResult: true})
	if err != nil {
		t.Fatal(err)
	}
	return server
}
