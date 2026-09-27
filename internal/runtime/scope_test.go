package runtime

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestScopeTree_CancelParentCancelsChildren(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		tree := NewScopeTree(context.Background())
		phase := domain.Scope{Machine: vocab.MachineSession, Epoch: 1}
		key := domain.Scope{Machine: vocab.MachineSession, Epoch: 1, Key: "u1"}
		phaseCtx := tree.Context(phase, domain.Scope{})
		keyCtx := tree.Context(key, phase)
		tree.Cancel(phase)
		synctest.Wait()
		if phaseCtx.Err() == nil || keyCtx.Err() == nil {
			t.Fatal("cancelling parent left a live context")
		}
		if got := tree.Scopes(); len(got) != 0 {
			t.Fatalf("active scopes after cancellation = %#v", got)
		}
	})
}

func TestScopeTree_NewEpochCancelsPreviousInstance(t *testing.T) {
	tree := NewScopeTree(context.Background())
	first := domain.Scope{Machine: vocab.MachineSession, Epoch: 1}
	second := domain.Scope{Machine: vocab.MachineSession, Epoch: 2}
	old := tree.Context(first, domain.Scope{})
	tree.Context(second, domain.Scope{})
	if old.Err() == nil {
		t.Fatal("old epoch remained live")
	}
	states := tree.Scopes()
	if len(states) != 1 || states[0].Scope != second {
		t.Fatalf("scopes = %#v", states)
	}
}

func TestScopeTree_CancelKeyPreservesSiblings(t *testing.T) {
	tree := NewScopeTree(context.Background())
	parent := domain.Scope{Machine: vocab.MachineSession, Epoch: 1}
	one := domain.Scope{Machine: vocab.MachineSession, Epoch: 1, Key: "one"}
	two := domain.Scope{Machine: vocab.MachineSession, Epoch: 1, Key: "two"}
	ctxOne := tree.Context(one, parent)
	ctxTwo := tree.Context(two, parent)
	tree.CancelKey(one)
	if ctxOne.Err() == nil || ctxTwo.Err() != nil {
		t.Fatal("key cancellation affected the wrong scope")
	}
	if got := len(tree.Scopes()); got != 2 {
		t.Fatalf("scope count = %d, want parent and sibling", got)
	}
}

func TestScopeTree_CloseCancelsRootAndDescendants(t *testing.T) {
	tree := NewScopeTree(context.Background())
	ctx := tree.Context(domain.Scope{Machine: vocab.MachineRun, Epoch: 1}, domain.Scope{})
	tree.Close()
	if ctx.Err() == nil {
		t.Fatal("Close left child live")
	}
}
