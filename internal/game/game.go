// Package game contains the pure DungeonFlux engine root.
package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

var _ ports.Engine = (*State)(nil)

// New creates a deterministic game state for one shot and seed.
func New(oneShot domain.OneShot, seed []byte, options ...Option) *State {
	return newState(oneShot, seed, options...)
}

// NewWithDebug creates a state with the debug event surface enabled. When
// debugStart is "combat", the state starts directly in Combat.
func NewWithDebug(oneShot domain.OneShot, seed []byte, debug bool, debugStart string, options ...Option) *State {
	return newDebugState(oneShot, seed, debug, debugStart, options...)
}

// Step applies one envelope and returns data effects. It performs no I/O and
// does not use the wall clock; the envelope supplies the logical time.
func (s *State) Step(env domain.Envelope) domain.StepOut {
	s.at = int64(env.At)
	if env.Event == nil {
		return s.rejected("missing_event")
	}
	if !s.accepts(env.Event) {
		return s.rejected("unaccepted_event")
	}
	s.version++
	return s.apply(env)
}

// LegalMoves returns a copy of the moves available to a seat in the current
// root state. Detailed phase moves are added by the phase machines.
func (s *State) LegalMoves(seat domain.SeatID) []vocab.MoveID {
	if seat != 1 && seat != 2 {
		return nil
	}
	if s.paused {
		return nil
	}
	moves := s.phase.LegalMoveViews(seat)
	ids := make([]vocab.MoveID, 0, len(moves))
	for _, move := range moves {
		if move.Enabled {
			ids = append(ids, move.ID)
		}
	}
	return ids
}

// View returns a read-only projection of the current state.
func (s *State) View() domain.View { return s.view() }

// Inspect returns a read-only debug summary of the current state.
func (s *State) Inspect() domain.Inspect {
	return domain.Inspect{
		Path:        string(s.path),
		Seed:        append([]byte(nil), s.seed...),
		DiceCounter: s.diceCounter,
		At:          time.Duration(s.at),
	}
}

func (s *State) accepts(event domain.Event) bool {
	if _, ok := event.(domain.HostCmd); ok {
		return true
	}
	if event.Kind() == vocab.EventDebugReset {
		return s.debug
	}
	return true
}

func (s *State) apply(env domain.Envelope) domain.StepOut {
	switch event := env.Event.(type) {
	case domain.HostCmd:
		return s.applyHost(event, env)
	case domain.DebugReset:
		return s.applyDebugReset(event, env)
	default:
		return s.applyPhase(env)
	}
}

func (s *State) applyHost(cmd domain.HostCmd, env domain.Envelope) domain.StepOut {
	if enabled, ok := timerToggleCommand(cmd.Cmd); ok {
		s.phase.SetTurnTimersEnabled(enabled)
	}
	if cmd.Cmd == vocab.HostForceD20 {
		out := s.applyForceD20(cmd.N)
		if out.Ack == nil || !out.Ack.Accepted {
			return out
		}
		if err := s.phase.ForceD20(cmd.N); err != nil {
			return s.rejected(err.Error())
		}
		return out
	}
	if cmd.Cmd == vocab.HostReset {
		previous := s.path
		s.resetPhase()
		effects := []domain.Effect{domain.NewRun{Seed: append([]byte(nil), s.seed...)}}
		effects = append(effects, s.phaseCueEffects(previous)...)
		return domain.StepOut{Effects: effects, Ack: acceptedAck(env)}
	}
	out := s.dispatch(domain.HostCmd{Cmd: cmd.Cmd})
	if out.Ack == nil || out.Ack.Reason == "" {
		out.Ack = acceptedAck(env)
	}
	if cmd.Cmd == vocab.HostPause {
		out.Effects = append(out.Effects, domain.PauseAll{})
	}
	if cmd.Cmd == vocab.HostResume {
		out.Effects = append(out.Effects, domain.ResumeAll{})
	}
	return out
}

func (s *State) applyPhase(env domain.Envelope) domain.StepOut {
	if join, ok := env.Event.(domain.Join); ok {
		if !s.applyJoin(join) {
			return s.rejected("invalid_seat")
		}
		return domain.StepOut{Ack: acceptedAck(env)}
	}
	out := s.dispatch(env.Event)
	if out.Ack == nil || out.Ack.Reason == "" {
		out.Ack = acceptedAck(env)
	}
	return out
}

func (s *State) dispatch(event domain.Event) domain.StepOut {
	previous := s.path
	result, err := s.phase.Step(event)
	if err != nil {
		return s.rejected("unaccepted_event")
	}
	s.path = s.phase.State()
	s.paused = s.phase.Paused()
	effects := append([]domain.Effect(nil), result.Effects...)
	effects = append(effects, s.phaseCueEffects(previous)...)
	return domain.StepOut{Effects: effects, Ack: &domain.Ack{Accepted: true}}
}

func (s *State) phaseCueEffects(previous vocab.StateID) []domain.Effect {
	if s.path == previous {
		return nil
	}
	return CueForState(s.path).Effects()
}

func (s *State) resetPhase() {
	defaultTimers := s.phase.DefaultTurnTimersEnabled()
	dispatcher, err := phase.NewWithSeed(s.oneShot, s.seed)
	if err == nil {
		dispatcher.ConfigureTurnTimers(defaultTimers)
		if s.debugStart != "" {
			_ = dispatcher.Goto(s.debugStart)
		}
		s.phase = dispatcher
	}
	s.path = s.phase.State()
	s.paused = false
	s.spotlight = 0
	s.nextD20 = 0
}

func (s *State) applyDebugReset(event domain.DebugReset, env domain.Envelope) domain.StepOut {
	previous := s.path
	s.seed = append(s.seed[:0], event.Seed...)
	s.resetPhase()
	effects := []domain.Effect{domain.NewRun{Seed: append([]byte(nil), s.seed...)}}
	effects = append(effects, s.phaseCueEffects(previous)...)
	return domain.StepOut{Effects: effects, Ack: acceptedAck(env)}
}

func (s *State) rejected(reason string) domain.StepOut {
	return domain.StepOut{Ack: &domain.Ack{Reason: reason}}
}

func acceptedAck(env domain.Envelope) *domain.Ack {
	if env.Reply == nil {
		return nil
	}
	return &domain.Ack{Accepted: true}
}
