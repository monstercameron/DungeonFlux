package fsm

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestChildren_routeDoneAndFailedToParent(t *testing.T) {
	children := NewChildren()
	parent := ScopeID{Machine: vocab.MachineSession}
	child := ScopeID{Machine: vocab.MachineCheck, Key: "1"}
	if err := children.Add(child, parent, vocab.MachineCheck); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		failed bool
		event  Event
		want   Event
	}{
		{name: "done", want: EventChildDone},
		{name: "failed", failed: true, want: EventChildFailed},
		{name: "specific event", event: "check_resolved", want: "check_resolved"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			routed, err := children.Route(ChildResult{Child: child, Failed: tc.failed, Event: tc.event, Value: "result"})
			if err != nil {
				t.Fatal(err)
			}
			if routed.Target != parent || routed.Source != child || routed.Event != tc.want || routed.Value != "result" {
				t.Fatalf("unexpected routed event: %#v", routed)
			}
		})
	}
}

func TestChildren_parentPointerAndLifecycle(t *testing.T) {
	children := NewChildren()
	parent := ScopeID{Machine: vocab.MachineRun}
	child := ScopeID{Machine: vocab.MachineCombat, Key: "1"}
	if err := children.Add(child, parent, vocab.MachineCombat); err != nil {
		t.Fatal(err)
	}
	registered, ok := children.Get(child)
	if !ok || registered.Parent != parent || registered.Machine != vocab.MachineCombat {
		t.Fatalf("unexpected child: %#v, %v", registered, ok)
	}
	if err := children.Remove(child); err != nil {
		t.Fatal(err)
	}
	if _, err := children.Route(ChildResult{Child: child}); err == nil {
		t.Fatal("late child result routed")
	}
}

func TestChildren_rejectsInvalidAndDuplicateEntries(t *testing.T) {
	children := NewChildren()
	parent := ScopeID{Machine: vocab.MachineSession}
	child := ScopeID{Machine: vocab.MachinePTT, Key: "2"}
	tests := []struct {
		name   string
		child  ScopeID
		parent ScopeID
		kind   vocab.MachineID
		want   string
	}{
		{name: "missing child", parent: parent, kind: vocab.MachinePTT, want: childParent},
		{name: "missing parent", child: child, kind: vocab.MachinePTT, want: childParent},
		{name: "missing machine", child: child, parent: parent, want: childParent},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := children.Add(tc.child, tc.parent, tc.kind); err == nil {
				t.Fatal("Add() accepted invalid child")
			} else {
				var childErr *ChildError
				if !errors.As(err, &childErr) || childErr.Reason != tc.want {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
	if err := children.Add(child, parent, vocab.MachinePTT); err != nil {
		t.Fatal(err)
	}
	if err := children.Add(child, parent, vocab.MachinePTT); err == nil {
		t.Fatal("duplicate child accepted")
	}
}
