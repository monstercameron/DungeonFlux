package fsm

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func testDef() Def {
	return Def{
		Initial: "idle",
		States:  []State{"idle", "ready"},
		Transitions: []Transition{
			{From: "idle", Event: "start", To: "ready"},
			{From: "ready", Event: "reset", To: "idle"},
		},
	}
}

func TestNew_rejectsInvalidDefinitions(t *testing.T) {
	tests := []struct {
		name string
		def  Def
	}{
		{name: "missing initial", def: Def{States: []State{"idle"}}},
		{name: "missing states", def: Def{Initial: "idle"}},
		{name: "unknown initial", def: Def{Initial: "missing", States: []State{"idle"}}},
		{name: "duplicate state", def: Def{Initial: "idle", States: []State{"idle", "idle"}}},
		{name: "unknown destination", def: Def{Initial: "idle", States: []State{"idle"}, Transitions: []Transition{{From: "idle", Event: "go", To: "done"}}}},
		{name: "duplicate transition", def: Def{Initial: "idle", States: []State{"idle"}, Transitions: []Transition{{From: "idle", Event: "go", To: "idle"}, {From: "idle", Event: "go", To: "idle"}}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New(tc.def)
			var rejection *Rejection
			if !errors.As(err, &rejection) || rejection.Reason != ReasonInvalidDefinition {
				t.Fatalf("New() error = %v, want invalid definition", err)
			}
		})
	}
}

func TestMachine_Step_transitionsAndRunsAction(t *testing.T) {
	def := testDef()
	def.Transitions[0].Action = func(event Event) []Effect {
		return []Effect{{Kind: vocab.EffectStartTimer, Name: string(event)}}
	}
	machine, err := New(def)
	if err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step("start")
	if err != nil {
		t.Fatal(err)
	}
	if result.From != "idle" || result.To != "ready" || result.Internal || machine.State() != "ready" {
		t.Fatalf("unexpected result: %#v, state %q", result, machine.State())
	}
	if len(result.Effects) != 1 || result.Effects[0].Kind != vocab.EffectStartTimer {
		t.Fatalf("unexpected effects: %#v", result.Effects)
	}
}

func TestMachine_Step_rejectsUnknownEventWithReason(t *testing.T) {
	machine, err := New(testDef())
	if err != nil {
		t.Fatal(err)
	}
	_, err = machine.Step("missing")
	var rejection *Rejection
	if !errors.As(err, &rejection) || rejection.Reason != ReasonUnknownEvent || rejection.State != "idle" || rejection.Event != "missing" {
		t.Fatalf("unexpected rejection: %v", err)
	}
}

func TestMachine_Step_rejectsGuardAndPreservesState(t *testing.T) {
	def := testDef()
	def.Transitions[0].Guard = func(Event) bool { return false }
	machine, err := New(def)
	if err != nil {
		t.Fatal(err)
	}
	_, err = machine.Step("start")
	var rejection *Rejection
	if !errors.As(err, &rejection) || rejection.Reason != ReasonGuardRejected || machine.State() != "idle" {
		t.Fatalf("unexpected rejection or state: %v, %q", err, machine.State())
	}
}

func TestMachine_Step_marksSelfTransitionInternal(t *testing.T) {
	def := Def{
		Initial:     "idle",
		States:      []State{"idle"},
		Transitions: []Transition{{From: "idle", Event: "refresh", To: "idle"}},
	}
	machine, err := New(def)
	if err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step("refresh")
	if err != nil || !result.Internal || machine.State() != "idle" {
		t.Fatalf("unexpected self-transition: %#v, %v", result, err)
	}
}
