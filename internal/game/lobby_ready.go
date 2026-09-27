package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func (s *State) lobbyStartReason() string {
	if s.paused {
		return "Resume the game before starting"
	}
	joined, ready := 0, 0
	for _, seat := range s.seats {
		if seat.Connected {
			joined++
			if seat.LobbyReady {
				ready++
			}
		}
	}
	if joined != 2 {
		return "Both players must join before starting. Use Skip to rehearse."
	}
	if ready != 2 {
		return "Both players must choose Ready before starting. Use Skip to rehearse."
	}
	return ""
}

// Lobby readiness is separate from locking a generated character in Creation.
// It survives a repeated Join, but never carries into a new run.
func (s *State) applyLobbyReady(act domain.Act, env domain.Envelope) domain.StepOut {
	if s.paused {
		return s.rejected("The game is paused")
	}
	if act.Move != vocab.MoveReady {
		return s.rejected("Choose Ready to join the adventure")
	}
	for index := range s.seats {
		seat := &s.seats[index]
		if seat.Seat != act.Seat || !seat.Connected {
			continue
		}
		if seat.LobbyReady {
			return s.rejected("You are already ready")
		}
		seat.LobbyReady = true
		return domain.StepOut{Ack: acceptedAck(env), Effects: UIAudioEffects(act)}
	}
	return s.rejected("Join the room before marking ready")
}

func (s *State) lobbyMoves(id domain.SeatID) []domain.MoveView {
	if s.paused {
		return nil
	}
	for _, seat := range s.seats {
		if seat.Seat != id || !seat.Connected {
			continue
		}
		move := domain.MoveView{ID: vocab.MoveReady, Label: "Ready", Enabled: !seat.LobbyReady}
		if seat.LobbyReady {
			move.Label, move.Reason = "Ready for adventure", "Waiting for the host to start"
		}
		return []domain.MoveView{move}
	}
	return nil
}
