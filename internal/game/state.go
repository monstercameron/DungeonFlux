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
	debug       bool
	debugStart  vocab.StateID
	paused      bool
	version     uint64
	at          int64
	spotlight   domain.SeatID
	diceCounter uint64
	nextD20     int
	seats       []domain.SeatView
}

func newState(oneShot domain.OneShot, seed []byte) *State {
	return newDebugState(oneShot, seed, false, "")
}

func newDebugState(oneShot domain.OneShot, seed []byte, debug bool, debugStart string) *State {
	seats := []domain.SeatView{
		{Seat: 1, PlayerNumber: 1},
		{Seat: 2, PlayerNumber: 2},
	}
	path := vocab.StateLobby
	if debug && debugStart == string(vocab.StateCombat) {
		path = vocab.StateCombat
	}
	return &State{
		oneShot:    oneShot,
		seed:       append([]byte(nil), seed...),
		path:       path,
		debug:      debug,
		debugStart: path,
		seats:      seats,
	}
}

func (s *State) view() domain.View {
	return domain.View{
		Version:   s.version,
		At:        time.Duration(s.at),
		Path:      s.path,
		Paused:    s.paused,
		Spotlight: s.spotlight,
		NextD20:   s.nextD20,
		Seats:     append([]domain.SeatView(nil), s.seats...),
	}
}
