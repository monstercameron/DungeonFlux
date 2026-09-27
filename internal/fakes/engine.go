package fakes

import (
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// EngineStep records an envelope passed to FakeEngine.Step.
type EngineStep struct {
	Envelope domain.Envelope
	Output   domain.StepOut
}

// FakeEngine is a scriptable engine. Step consumes StepOuts in order.
type FakeEngine struct {
	Steps        []domain.StepOut
	ViewValue    domain.View
	InspectValue domain.Inspect
	Legal        map[domain.SeatID][]vocab.MoveID
	Calls        []EngineStep
	mu           sync.Mutex
}

// Step records env and returns the next scripted output, or a zero output.
func (f *FakeEngine) Step(env domain.Envelope) domain.StepOut {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := domain.StepOut{}
	if len(f.Steps) > 0 {
		out, f.Steps = f.Steps[0], f.Steps[1:]
	}
	f.Calls = append(f.Calls, EngineStep{env, out})
	return out
}

// LegalMoves returns a copy of the configured legal moves for seat.
func (f *FakeEngine) LegalMoves(seat domain.SeatID) []vocab.MoveID {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]vocab.MoveID(nil), f.Legal[seat]...)
}

// View returns a deep copy of the configured view.
func (f *FakeEngine) View() domain.View {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.ViewValue.DeepCopy()
}

// Inspect returns the configured inspection snapshot.
func (f *FakeEngine) Inspect() domain.Inspect {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.InspectValue
}
