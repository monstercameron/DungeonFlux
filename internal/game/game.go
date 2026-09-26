// Package game contains the pure DungeonFlux engine root.
package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

var _ ports.Engine = (*State)(nil)

// New creates a deterministic game state for one shot and seed.
func New(oneShot domain.OneShot, seed []byte) *State {
	return newState(oneShot, seed)
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
	switch s.path {
	case vocab.StateLobby:
		return []vocab.MoveID{vocab.MoveReady}
	default:
		return nil
	}
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
	switch event.Kind() {
	case vocab.EventDebugReset:
		return true
	default:
		return false
	}
}

func (s *State) apply(env domain.Envelope) domain.StepOut {
	switch event := env.Event.(type) {
	case domain.HostCmd:
		return s.applyHost(event)
	case domain.DebugReset:
		s.seed = append(s.seed[:0], event.Seed...)
		s.path = vocab.StateLobby
		s.paused = false
		s.spotlight = 0
		return domain.StepOut{Effects: []domain.Effect{domain.NewRun{Seed: append([]byte(nil), s.seed...)}}}
	default:
		return domain.StepOut{Ack: acceptedAck(env)}
	}
}

func (s *State) applyHost(cmd domain.HostCmd) domain.StepOut {
	switch cmd.Cmd {
	case vocab.HostStart:
		if s.path != vocab.StateLobby {
			return s.rejected("already_started")
		}
		s.path = vocab.StateCreation
		return domain.StepOut{
			Effects: []domain.Effect{domain.StartTimer{
				Name: "creation_timeout", After: 30 * time.Second,
				Pausable: true, Scope: domain.Scope{Machine: vocab.MachineSession},
			}},
			Ack: &domain.Ack{Accepted: true},
		}
	case vocab.HostPause:
		s.paused = true
		return domain.StepOut{Effects: []domain.Effect{domain.PauseAll{}}, Ack: &domain.Ack{Accepted: true}}
	case vocab.HostResume:
		s.paused = false
		return domain.StepOut{Effects: []domain.Effect{domain.ResumeAll{}}, Ack: &domain.Ack{Accepted: true}}
	case vocab.HostReset:
		s.path = vocab.StateLobby
		s.paused = false
		return domain.StepOut{Effects: []domain.Effect{domain.NewRun{Seed: append([]byte(nil), s.seed...)}}, Ack: &domain.Ack{Accepted: true}}
	default:
		return s.rejected("unsupported_host_command")
	}
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
