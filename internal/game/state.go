package game

import (
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// State is the pure, single-threaded state owned by one game room.
type State struct {
	oneShot          domain.OneShot
	seed             []byte
	path             vocab.StateID
	debug            bool
	debugStart       vocab.StateID
	paused           bool
	version          uint64
	at               int64
	spotlight        domain.SeatID
	diceCounter      uint64
	nextD20          int
	seats            []domain.SeatView
	lobby            Lobby
	phase            phase.Machine
	narrationSpeaker string
	narrationText    string
	narrationLineID  domain.UtteranceID
	narrationDone    bool
}

func newState(oneShot domain.OneShot, seed []byte, options ...Option) *State {
	return newDebugState(oneShot, seed, false, "", options...)
}

func newDebugState(oneShot domain.OneShot, seed []byte, debug bool, debugStart string, options ...Option) *State {
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
		_, _ = dispatcher.DebugGoto(vocab.StateID(debugStart))
		path = dispatcher.State()
	}
	state := &State{
		oneShot:    oneShot,
		seed:       append([]byte(nil), seed...),
		path:       path,
		debug:      debug,
		debugStart: path,
		seats:      seats,
		phase:      dispatcher,
	}
	for _, option := range options {
		if option != nil {
			option(state)
		}
	}
	state.lobby.Seats = state.lobbySeats()
	return state
}

func (s *State) view() domain.View {
	view := s.phase.View()
	view.Version = s.version
	view.At = time.Duration(s.at)
	view.Path = s.path
	view.Paused = s.paused
	view.NextD20 = s.nextD20
	view.Seats = mergeSeatViews(view.Seats, s.seats)
	view.Scene.Narration = s.narrationText
	view.Scene.NarrationSpeaker = s.narrationSpeaker
	view.Scene.NarrationLineID = s.narrationLineID
	view.Scene.NarrationDone = s.narrationDone
	return view
}

func (s *State) applyJoin(join domain.Join) bool {
	for index := range s.seats {
		if s.seats[index].Seat != join.Seat {
			continue
		}
		s.seats[index].Connected = true
		s.seats[index].Locale = join.Locale
		s.seats[index].PlayerName = join.Name
		s.lobby.Seats = s.lobbySeats()
		return true
	}
	return false
}

func mergeSeatViews(current, joined []domain.SeatView) []domain.SeatView {
	if len(current) == 0 {
		return append([]domain.SeatView(nil), joined...)
	}
	out := append([]domain.SeatView(nil), current...)
	for index := range out {
		for _, seat := range joined {
			if seat.Seat != out[index].Seat {
				continue
			}
			out[index].Connected = seat.Connected
			out[index].PlayerName = seat.PlayerName
			out[index].Locale = seat.Locale
			break
		}
	}
	return out
}
