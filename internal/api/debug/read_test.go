package debug

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func testServer(t *testing.T) *Server {
	t.Helper()
	engine := &fakes.FakeEngine{
		ViewValue:    domain.View{Version: 4, Path: vocab.StateConversation, Spotlight: 2, Callout: "look", Preload: []string{"a"}, Slots: []domain.SlotView{{Name: "hero", State: "ready"}}},
		InspectValue: domain.Inspect{Path: "conversation", Seed: []byte{1, 2}, DiceCounter: 3, Scopes: []domain.ScopeState{{State: "active"}}},
		Legal:        map[domain.SeatID][]vocab.MoveID{2: {vocab.MovePersuade}},
	}
	server, err := NewServer(engine, &fakes.FakeInbox{PostResult: true})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

func TestReads_ReturnSnapshotsAndValidation(t *testing.T) {
	server := testServer(t)
	ctx := context.Background()
	state, err := server.State(ctx, &df.DebugRoom{Room: "room"})
	if err != nil || state.GetPhase() != "conversation" || state.GetSeed() != "0102" || state.GetDiceCounter() != 3 {
		t.Fatalf("state = %+v, err = %v", state, err)
	}
	view, err := server.View(ctx, &df.ViewRequest{View: "dm"})
	if err != nil || view.GetVersion() != 4 || view.GetDm().GetCallout() != "look" {
		t.Fatalf("view = %+v, err = %v", view, err)
	}
	legal, err := server.Legal(ctx, &df.SeatRequest{Seat: "2"})
	if err != nil || len(legal.GetMoves()) != 1 || legal.GetMoves()[0].GetMoveId() != "persuade" {
		t.Fatalf("legal = %+v, err = %v", legal, err)
	}
	assets, err := server.Assets(ctx, &df.DebugRoom{Room: "room"})
	if err != nil || len(assets.GetSlots()) != 1 {
		t.Fatalf("assets = %+v, err = %v", assets, err)
	}
	if _, err := server.State(ctx, nil); err == nil {
		t.Fatal("nil state request accepted")
	}
	if _, err := server.Legal(ctx, &df.SeatRequest{Seat: "x"}); err == nil {
		t.Fatal("invalid seat accepted")
	}
}

func TestReads_EmptyOperationalSurfaces(t *testing.T) {
	server := testServer(t)
	ctx := context.Background()
	if got, err := server.Scopes(ctx, &df.DebugRoom{}); err != nil || got.GetScopesJson() == "" {
		t.Fatalf("scopes = %+v, err = %v", got, err)
	}
	if got, err := server.Clients(ctx, &df.DebugRoom{}); err != nil || got == nil {
		t.Fatalf("clients = %+v, err = %v", got, err)
	}
	if got, err := server.Costs(ctx, &df.DebugRoom{}); err != nil || got == nil {
		t.Fatalf("costs = %+v, err = %v", got, err)
	}
	if err := server.Events(nil, nil); err == nil {
		t.Fatal("nil events stream accepted")
	}
	if err := server.Logs(nil, nil); err == nil {
		t.Fatal("nil logs stream accepted")
	}
}

func TestView_ChoosesRequestedProjection(t *testing.T) {
	server := testServer(t)
	for _, requested := range []string{"phone", "host", "", "unknown"} {
		view, err := server.View(context.Background(), &df.ViewRequest{View: requested})
		if err != nil || view == nil {
			t.Fatalf("view %q = %+v, err = %v", requested, view, err)
		}
	}
	if _, err := server.Assets(context.Background(), nil); err == nil {
		t.Fatal("nil assets request accepted")
	}
	if _, err := server.Clients(context.Background(), nil); err == nil {
		t.Fatal("nil clients request accepted")
	}
}
