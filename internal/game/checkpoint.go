package game

import "github.com/monstercameron/DungeonFlux/internal/domain"

func (s *State) applyCheckpoint(event domain.DebugCheckpoint, env domain.Envelope) domain.StepOut {
	if event.Operation != "save" && event.Operation != "load" {
		return s.rejected("checkpoint operation must be save or load")
	}
	if len(event.Name) == 0 || len(event.Name) > 64 {
		return s.rejected("checkpoint name must contain 1 to 64 letters, digits, underscores or hyphens")
	}
	for _, c := range event.Name {
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return s.rejected("checkpoint name must contain only letters, digits, underscores or hyphens")
		}
	}
	return domain.StepOut{Ack: acceptedAck(env), Effects: []domain.Effect{domain.Checkpoint(event)}}
}
