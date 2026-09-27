package game

import (
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const hostSplatOn vocab.HostCmd = "SPLAT_ON"

// WithSplat configures the preferred renderer for a newly created run.
// A scene must exist before an enabled preference can take effect.
func WithSplat(enabled bool) Option {
	return func(state *State) { state.splatEnabled = enabled && state.splatAvailable() }
}

func (s *State) splatAvailable() bool {
	return strings.TrimSpace(s.oneShot.Encounter.Battlefield.SceneURL) != ""
}

func (s *State) applySplatToggle(command vocab.HostCmd, env domain.Envelope) (domain.StepOut, bool) {
	if command != vocab.HostSplatOff && command != hostSplatOn {
		return domain.StepOut{}, false
	}
	if command == hostSplatOn && !s.splatAvailable() {
		return s.rejected("battlefield_unavailable"), true
	}
	s.splatEnabled = command == hostSplatOn
	return domain.StepOut{Ack: acceptedAck(env)}, true
}
