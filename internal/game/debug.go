package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func (s *State) applyForceD20(value int) domain.StepOut {
	if value < 1 || value > 20 {
		return s.rejected("d20_must_be_between_1_and_20")
	}
	s.nextD20 = value
	return domain.StepOut{Ack: &domain.Ack{Accepted: true}}
}

func (s *State) consumeForcedD20() (int, bool) {
	if s.nextD20 == 0 {
		return 0, false
	}
	value := s.nextD20
	s.nextD20 = 0
	return value, true
}
