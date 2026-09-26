package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
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
	phase       phase.Machine
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
	dispatcher, err := phase.NewWithSeed(oneShot, seed)
	if err != nil {
		dispatcher, _ = phase.New()
	}
	if debug && debugStart != "" {
		_ = dispatcher.Goto(vocab.StateID(debugStart))
		path = dispatcher.State()
	}
	return &State{
		oneShot:    oneShot,
		seed:       append([]byte(nil), seed...),
		path:       path,
		debug:      debug,
		debugStart: path,
		seats:      seats,
		phase:      dispatcher,
	}
}

func (s *State) view() domain.View {
	view := s.phase.View()
	view.Version = s.version
	view.At = time.Duration(s.at)
	view.Path = s.path
	view.Paused = s.paused
	view.NextD20 = s.nextD20
	if view.Seats == nil {
		view.Seats = append([]domain.SeatView(nil), s.seats...)
	}
	return view
}
