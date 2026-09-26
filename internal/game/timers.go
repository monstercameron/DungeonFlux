package game

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const hostTimersOn vocab.HostCmd = "TIMERS_ON"

// WithTurnTimers configures whether a new run starts with turn timers on.
func WithTurnTimers(enabled bool) Option {
	return func(state *State) {
		state.phase.ConfigureTurnTimers(enabled)
	}
}

// TurnTimersEnabled reports whether the current run may start turn timers.
func (s *State) TurnTimersEnabled() bool {
	if s == nil {
		return false
	}
	return s.phase.TurnTimersEnabled()
}

func timerToggleCommand(cmd vocab.HostCmd) (bool, bool) {
	switch cmd {
	case vocab.HostTimersOff:
		return false, true
	case hostTimersOn:
		return true, true
	default:
		return false, false
	}
}
