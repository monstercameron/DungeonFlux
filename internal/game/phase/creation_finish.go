package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/creation"
)

// finishCreation applies the same fallback as the deadline, including when
// automatic timers are disabled. A host Skip still finalizes both heroes.
func (m *Machine) finishCreation() (Result, error) {
	result, err := m.creation.Step(domain.TimerFired{Name: "creation_timeout"})
	if err != nil {
		return Result{}, err
	}
	return m.applyCreationResult(result)
}

func (m *Machine) applyCreationResult(result creation.Result) (Result, error) {
	// A bulk deadline changes both seats and has no single Result.Seat.
	for _, seat := range m.creation.Seats() {
		m.updateCreationSeat(seat)
	}
	if !result.Complete {
		return Result{Effects: result.Effects}, nil
	}
	result.Effects = append(result.Effects,
		domain.CancelTimer{Name: "seat_deadline:1"},
		domain.CancelTimer{Name: "seat_deadline:2"},
	)
	return m.transition(eventCreationEnd, result.Effects)
}
