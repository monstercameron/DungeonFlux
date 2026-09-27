package api

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func seatReady(seat domain.SeatView) bool {
	for _, move := range seat.Moves {
		if move.ID == vocab.MoveReady {
			return characterLocked(seat.Moves)
		}
	}
	return seat.Build != nil || seat.Character != nil
}
