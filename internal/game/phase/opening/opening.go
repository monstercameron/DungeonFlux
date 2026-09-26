package opening

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// OpeningCannedAsset is the build-time fallback audio for the opening line.
const OpeningCannedAsset domain.AssetID = "canned_opening"

// State identifies the lifecycle of an opening phase.
type State string

const (
	// StateReady means the phase has not started.
	StateReady State = "ready"
	// StatePlaying means the opening line is in progress.
	StatePlaying State = "playing"
)

// Result contains the next phase state and effects for the runtime.
type Result struct {
	State   State
	Effects []domain.Effect
}

// Machine is the pure Opening phase machine.
type Machine struct {
	state State
}

// New creates an Opening machine ready to play the canned opening line.
func New() Machine {
	return Machine{state: StateReady}
}

// State reports the current opening lifecycle state.
func (m Machine) State() State {
	return m.state
}

// Enter starts the opening phase and emits the canned opening audio.
func (m *Machine) Enter() Result {
	if m.state != StateReady {
		return Result{State: m.state}
	}
	m.state = StatePlaying
	return Result{
		State:   m.state,
		Effects: []domain.Effect{domain.PlayCanned{AssetID: OpeningCannedAsset}},
	}
}

// Step advances the opening phase. A completed line returns to the caller's
// dispatcher; other events are ignored so late callbacks cannot add effects.
func (m *Machine) Step(event domain.Event) Result {
	if m == nil || m.state != StatePlaying || event == nil {
		return Result{State: m.state}
	}
	if event.Kind() != vocab.EventLineDone {
		return Result{State: m.state}
	}
	m.state = StateReady
	return Result{State: m.state}
}
