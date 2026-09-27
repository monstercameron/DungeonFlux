package game

import (
	"fmt"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// isDebugEvent identifies events that are accepted only by a debug-enabled
// room. Host commands remain available in every room.
func isDebugEvent(event domain.Event) bool {
	switch event.(type) {
	case domain.DebugGoto, domain.DebugPatch, domain.DebugTimer, domain.DebugForceDice, domain.DebugReset, domain.DebugCheckpoint:
		return true
	default:
		return false
	}
}

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

func (s *State) applyDebugGoto(event domain.DebugGoto, env domain.Envelope) domain.StepOut {
	dispatcher, err := phase.NewWithSeed(s.oneShot, s.seed)
	if err != nil {
		return s.rejected(err.Error())
	}
	dispatcher.ConfigureCombatMoveUI(s.phase.CombatMoveUI())
	dispatcher.ConfigureTurnTimers(s.phase.TurnTimersEnabled())
	result, err := dispatcher.DebugGoto(event.Phase)
	if err != nil {
		return s.rejected(err.Error())
	}
	previous := s.path
	s.phase = dispatcher
	s.path = dispatcher.State()
	s.paused = false
	s.clearNarration()
	effects := debugCancelEffects()
	effects = append(effects, result.Effects...)
	effects = append(effects, s.phaseCueEffects(previous)...)
	s.applyNarrationEffects(effects)
	return domain.StepOut{Effects: effects, Ack: acceptedAck(env)}
}

func (s *State) applyDebugPatch(event domain.DebugPatch, env domain.Envelope) domain.StepOut {
	result, err := s.phase.DebugPatch(event.Target, event.Fields)
	if err != nil {
		return s.rejected(err.Error())
	}
	previous := s.path
	s.path = s.phase.State()
	s.paused = s.phase.Paused()
	effects := append([]domain.Effect(nil), result.Effects...)
	if s.path != previous {
		s.clearNarration()
		effects = append(effects, s.phaseCueEffects(previous)...)
	}
	s.applyNarrationEffects(effects)
	return domain.StepOut{Effects: effects, Ack: acceptedAck(env)}
}

func (s *State) applyDebugTimer(event domain.DebugTimer, env domain.Envelope) domain.StepOut {
	if strings.TrimSpace(event.Name) == "" {
		return s.rejected("timer name is required")
	}
	switch strings.ToLower(strings.TrimSpace(event.Op)) {
	case "fire", "expire":
		out := s.dispatch(domain.TimerFired{Name: event.Name})
		if out.Ack != nil && out.Ack.Accepted {
			out.Ack = acceptedAck(env)
		}
		return out
	case "cancel":
		return domain.StepOut{Effects: []domain.Effect{domain.CancelTimer{Name: event.Name}}, Ack: acceptedAck(env)}
	case "set":
		if event.MS < 0 {
			return s.rejected("timer milliseconds must be non-negative")
		}
		return domain.StepOut{Effects: []domain.Effect{domain.StartTimer{
			Name: event.Name, After: time.Duration(event.MS) * time.Millisecond, Pausable: true,
			Scope: domain.Scope{Machine: vocab.MachineSession},
		}}, Ack: acceptedAck(env)}
	case "pause":
		return domain.StepOut{Effects: []domain.Effect{domain.FreezeTimer{Name: event.Name}}, Ack: acceptedAck(env)}
	case "resume", "thaw":
		return domain.StepOut{Effects: []domain.Effect{domain.ThawTimer{Name: event.Name}}, Ack: acceptedAck(env)}
	default:
		return s.rejected(fmt.Sprintf("unknown timer operation %q", event.Op))
	}
}

func debugCancelEffects() []domain.Effect {
	effects := []domain.Effect{
		domain.CancelScope{Scope: domain.Scope{Machine: vocab.MachineSession}},
		domain.CancelScope{Scope: domain.Scope{Machine: vocab.MachineCheck}},
		domain.CancelScope{Scope: domain.Scope{Machine: vocab.MachineCombat}},
	}
	for _, name := range []string{
		"creation_timeout", "seat_deadline:1", "seat_deadline:2", "turn_timer",
		"roll_resolved", "combat_turn_timer", "combat_cap", "combat_dash_resolved",
	} {
		effects = append(effects, domain.CancelTimer{Name: name})
	}
	return effects
}
