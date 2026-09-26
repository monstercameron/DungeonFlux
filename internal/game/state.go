package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State is the pure, single-threaded state owned by one game room.
type State struct {
	oneShot     domain.OneShot
	seed        []byte
	path        vocab.StateID
	paused      bool
	version     uint64
	at          int64
	spotlight   domain.SeatID
	diceCounter uint64
	seats       []domain.SeatView
}

func newState(oneShot domain.OneShot, seed []byte) *State {
	seats := []domain.SeatView{
		{Seat: 1, PlayerNumber: 1},
		{Seat: 2, PlayerNumber: 2},
	}
	return &State{
		oneShot: oneShot,
		seed:    append([]byte(nil), seed...),
		path:    vocab.StateLobby,
		seats:   seats,
	}
}

func (s *State) view() domain.View {
	return domain.View{
		Version:   s.version,
		At:        time.Duration(s.at),
		Path:      s.path,
		Paused:    s.paused,
		Spotlight: s.spotlight,
		Seats:     append([]domain.SeatView(nil), s.seats...),
	}
}
