package ports

import (
	"context"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestResolveCallMeta_PreservesRequestAndCopiesPosition(t *testing.T) {
	requested := CallMeta{Role: vocab.RoleNPCReply, Locale: "es", ForceReplay: true, UtteranceID: "actual"}
	if got := ResolveCallMeta(context.Background(), requested); got != requested {
		t.Fatalf("unbound request changed: %+v", got)
	}
	bound := CallMeta{Run: "run", Phase: vocab.StateConversation, Seat: 2, Index: 3, Role: vocab.RoleInterpret, UtteranceID: "bound", Speculative: true}
	ctx := WithCallMeta(context.Background(), bound)
	got := ResolveCallMeta(ctx, requested)
	if got.Run != "run" || got.Phase != vocab.StateConversation || got.Seat != 2 || got.Index != 3 || got.Role != requested.Role || got.Locale != "es" || !got.ForceReplay || !got.Speculative || got.UtteranceID != "actual" {
		t.Fatalf("resolved request=%+v", got)
	}
	bound.Index = 99
	if got := ResolveCallMeta(ctx, CallMeta{}); got.Index != 3 || got.Role != vocab.RoleInterpret || got.UtteranceID != "bound" {
		t.Fatalf("bound metadata was not a detached value: %+v", got)
	}
	if got := ResolveCallMeta(WithCallMeta(ctx, CallMeta{}), CallMeta{Seat: 1}); got.Seat != 1 {
		t.Fatal("unassigned work seat erased request seat")
	}
}
