// Package fsm provides pure, table-driven state machines for the game engine.
// It depends only on the vocabulary package and has no I/O or concurrency.
package fsm

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State identifies a state in a machine definition.
type State = vocab.StateID

// Event identifies an input accepted by a machine.
type Event = vocab.EventKind

// Effect is a pure value returned by a transition action for the runner.
type Effect struct {
	Kind vocab.EffectKind
	Name string
	Data any
}

// Guard decides whether a transition can accept an event.
type Guard func(Event) bool

// Action computes effects after a transition is accepted.
type Action func(Event) []Effect

// Transition describes one event edge from a state. A transition with the
// same source and destination is an internal self-transition.
type Transition struct {
	From   State
	Event  Event
	To     State
	Guard  Guard
	Action Action
}

// Def describes the states and transitions of a machine.
type Def struct {
	Initial     State
	States      []State
	Transitions []Transition
}

// Result describes an accepted event and its computed effects.
type Result struct {
	From     State
	To       State
	Event    Event
	Internal bool
	Effects  []Effect
}

// Reason explains why an event was rejected.
type Reason string

const (
	// ReasonUnknownEvent means no transition exists for the current state.
	ReasonUnknownEvent Reason = "unknown_event"
	// ReasonGuardRejected means a matching transition's guard returned false.
	ReasonGuardRejected Reason = "guard_rejected"
	// ReasonInvalidDefinition means a definition is malformed.
	ReasonInvalidDefinition Reason = "invalid_definition"
)

// Rejection is a structured machine error that callers can inspect without
// parsing an error string.
type Rejection struct {
	State  State
	Event  Event
	Reason Reason
}

// Error implements error for a rejected event or invalid definition.
func (e *Rejection) Error() string {
	return fmt.Sprintf("fsm rejected event %q in state %q: %s", e.Event, e.State, e.Reason)
}

// Machine is a state machine instance created from a validated definition.
type Machine struct {
	def   Def
	state State
}

// New validates a definition and returns a machine at its initial state.
func New(def Def) (Machine, error) {
	if def.Initial == "" {
		return Machine{}, &Rejection{Reason: ReasonInvalidDefinition}
	}
	if len(def.States) == 0 {
		return Machine{}, &Rejection{State: def.Initial, Reason: ReasonInvalidDefinition}
	}
	known := make(map[State]struct{}, len(def.States))
	for _, state := range def.States {
		if state == "" {
			return Machine{}, &Rejection{Reason: ReasonInvalidDefinition}
		}
		if _, exists := known[state]; exists {
			return Machine{}, &Rejection{State: state, Reason: ReasonInvalidDefinition}
		}
		known[state] = struct{}{}
	}
	if _, exists := known[def.Initial]; !exists {
		return Machine{}, &Rejection{State: def.Initial, Reason: ReasonInvalidDefinition}
	}
	seen := make(map[[2]string]struct{}, len(def.Transitions))
	for _, transition := range def.Transitions {
		if transition.From == "" || transition.Event == "" || transition.To == "" {
			return Machine{}, &Rejection{State: transition.From, Event: transition.Event, Reason: ReasonInvalidDefinition}
		}
		if _, exists := known[transition.From]; !exists {
			return Machine{}, &Rejection{State: transition.From, Event: transition.Event, Reason: ReasonInvalidDefinition}
		}
		if _, exists := known[transition.To]; !exists {
			return Machine{}, &Rejection{State: transition.To, Event: transition.Event, Reason: ReasonInvalidDefinition}
		}
		key := [2]string{string(transition.From), string(transition.Event)}
		if _, exists := seen[key]; exists {
			return Machine{}, &Rejection{State: transition.From, Event: transition.Event, Reason: ReasonInvalidDefinition}
		}
		seen[key] = struct{}{}
	}
	return Machine{def: def, state: def.Initial}, nil
}

// State returns the machine's current state.
func (m Machine) State() State { return m.state }

// Step accepts one event, returning its transition result or a rejection.
func (m *Machine) Step(event Event) (Result, error) {
	for _, transition := range m.def.Transitions {
		if transition.From != m.state || transition.Event != event {
			continue
		}
		if transition.Guard != nil && !transition.Guard(event) {
			return Result{}, &Rejection{State: m.state, Event: event, Reason: ReasonGuardRejected}
		}
		from := m.state
		m.state = transition.To
		var effects []Effect
		if transition.Action != nil {
			effects = transition.Action(event)
		}
		return Result{From: from, To: m.state, Event: event, Internal: from == m.state, Effects: effects}, nil
	}
	return Result{}, &Rejection{State: m.state, Event: event, Reason: ReasonUnknownEvent}
}
