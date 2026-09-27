package runtime

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestCallSequence_AssignsBeforeWorkersComplete(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := &checkpointEngine{view: domain.View{Path: vocab.StateConversation, Spotlight: 2}}
		room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
		defer room.scopes.Close()
		calls := make(chan ports.CallMeta, 4)
		Handle[domain.StartLine](room.runner.(*Runner), func(ctx context.Context, _ domain.StartLine, _ domain.Scope, _ ports.Inbox) {
			calls <- ports.ResolveCallMeta(ctx, ports.CallMeta{})
			<-ctx.Done()
		})
		if err := room.applyEffects(context.Background(), []domain.Effect{
			domain.StartLine{Role: vocab.RoleNPCReply, UtteranceID: "first"},
			domain.StartLine{Role: vocab.RoleNPCReply, UtteranceID: "second"},
		}, domain.Scope{}); err != nil {
			t.Fatal(err)
		}
		seen := map[domain.UtteranceID]ports.CallMeta{}
		for range 2 {
			meta := <-calls
			seen[meta.UtteranceID] = meta
		}
		if seen["first"].Index != 0 || seen["second"].Index != 1 || seen["second"].Phase != vocab.StateConversation || seen["second"].Seat != 2 {
			t.Fatalf("dispatch positions=%+v", seen)
		}
	})
}

func TestCallSequence_SeparatesRolesSeatsAndPhaseEntries(t *testing.T) {
	engine := &checkpointEngine{view: domain.View{Path: vocab.StateCreation}}
	room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
	defer room.scopes.Close()
	for index, tc := range []struct {
		effect domain.Effect
		want   int
	}{
		{domain.CharacterFlavor{Seat: 1}, 0}, {domain.CharacterFlavor{Seat: 2}, 0},
		{domain.CharacterFlavor{Seat: 1}, 1}, {domain.Interpret{Seat: 1}, 0},
	} {
		call, ok := room.reserveCall(context.Background(), tc.effect, uint64(index+1))
		if !ok || call.meta.Index != tc.want {
			t.Fatalf("call %d=%+v", index, call)
		}
	}
	for _, phase := range []vocab.StateID{vocab.StateConversation, vocab.StateCreation} {
		engine.view.Path = phase
		call, _ := room.reserveCall(context.Background(), domain.CharacterFlavor{Seat: 1}, 9)
		if call.meta.Index != 0 || call.meta.Phase != phase {
			t.Fatalf("phase entry=%+v", call)
		}
	}
	if _, ok := room.reserveCall(context.Background(), domain.PlaySound{}, 10); ok {
		t.Fatal("sound consumed a model position")
	}
	meta, ok := effectCallMeta(domain.PrerenderText{Role: vocab.RoleCombatOutcomes}, domain.View{})
	if !ok || !meta.Speculative || meta.Role != vocab.RoleCombatOutcomes {
		t.Fatalf("prerender=%+v", meta)
	}
}

func TestCallSequence_CancellationReusesOnlyItsOwnReservation(t *testing.T) {
	engine := &checkpointEngine{view: domain.View{Path: vocab.StateOpening}}
	room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
	defer room.scopes.Close()
	ctx, cancel := context.WithCancel(context.Background())
	effect := domain.StartLine{Role: vocab.RoleOpening, Hold: true}
	first, _ := room.reserveCall(ctx, effect, 1)
	room.pending = map[uint64]pendingWork{1: {ctx: ctx, call: first, hasCall: true}}
	second, _ := room.reserveCall(context.Background(), effect, 2)
	cancel()
	third, _ := room.reserveCall(context.Background(), effect, 3)
	fourth, _ := room.reserveCall(context.Background(), effect, 4)
	if first.meta.Index != 0 || !first.meta.Speculative || second.meta.Index != 1 || third.meta.Index != 0 || fourth.meta.Index != 2 {
		t.Fatalf("positions=%d,%d,%d,%d", first.meta.Index, second.meta.Index, third.meta.Index, fourth.meta.Index)
	}
	room.invalidateWork(context.Background())
	reset, _ := room.reserveCall(context.Background(), effect, 5)
	if reset.meta.Index != 0 {
		t.Fatalf("reset index=%d", reset.meta.Index)
	}
}

func TestCallSequence_CheckpointRestartsSamePositionAndRestoresNext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := &checkpointEngine{view: domain.View{Path: vocab.StateConversation, Spotlight: 1}}
		room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
		defer room.scopes.Close()
		calls := make(chan ports.CallMeta, 8)
		Handle[domain.StartLine](room.runner.(*Runner), func(ctx context.Context, _ domain.StartLine, _ domain.Scope, _ ports.Inbox) {
			calls <- ports.ResolveCallMeta(ctx, ports.CallMeta{})
			<-ctx.Done()
		})
		effect := domain.StartLine{Role: vocab.RoleNPCReply, UtteranceID: "line"}
		if err := room.applyEffects(context.Background(), []domain.Effect{effect}, domain.Scope{}); err != nil {
			t.Fatal(err)
		}
		if first := <-calls; first.Index != 0 {
			t.Fatalf("first=%+v", first)
		}
		checkpointCommand(t, room, "save", "conversation", true)
		for range 2 {
			checkpointCommand(t, room, "load", "conversation", true)
			if restored := <-calls; restored.Index != 0 || restored.Phase != vocab.StateConversation {
				t.Fatalf("restored=%+v", restored)
			}
			if err := room.applyEffects(context.Background(), []domain.Effect{effect}, domain.Scope{}); err != nil {
				t.Fatal(err)
			}
			if next := <-calls; next.Index != 1 {
				t.Fatalf("next=%+v", next)
			}
		}
	})
}

func TestCallSequence_CompletedCallRetainsPositionAfterScopeCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		engine := &checkpointEngine{view: domain.View{Path: vocab.StateConversation}}
		room := NewRoom(engine, clock.NewFake(time.Unix(0, 0)), nil, nil, nil)
		defer room.scopes.Close()
		effect := domain.StartLine{Role: vocab.RoleNPCReply}
		Handle[domain.StartLine](room.runner.(*Runner), func(context.Context, domain.StartLine, domain.Scope, ports.Inbox) {})
		if err := room.applyEffects(context.Background(), []domain.Effect{effect}, domain.Scope{}); err != nil {
			t.Fatal(err)
		}
		synctest.Wait()
		if !room.pending[1].completed.Load() {
			t.Fatal("worker did not complete")
		}
		room.scopes.Close()
		next, _ := room.reserveCall(context.Background(), effect, 2)
		if next.meta.Index != 1 {
			t.Fatalf("completed position reused: %d", next.meta.Index)
		}
	})
}
