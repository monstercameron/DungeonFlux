package game

import "github.com/monstercameron/DungeonFlux/internal/domain"

// Lobby contains room metadata supplied by the composition root and the
// room-level seat state retained across phase and run resets.
type Lobby struct {
	RoomCode string
	JoinURL  string
	QRAsset  domain.AssetID
	Seats    []LobbySeat
}

// LobbySeat is the engine's room-level join record for one player seat.
type LobbySeat struct {
	Seat         domain.SeatID
	PlayerNumber int
	Joined       bool
	Name         string
	Locale       string
}

// Option configures a newly constructed game state.
type Option func(*State)

// WithLobby supplies the room code, phone URL, and QR asset reference shown
// by the lobby. The metadata remains attached when the run is reset.
func WithLobby(lobby Lobby) Option {
	return func(state *State) {
		state.lobby = cloneLobby(lobby)
	}
}

// Lobby returns a detached copy of the room-level lobby data.
func (s *State) Lobby() Lobby {
	if s == nil {
		return Lobby{}
	}
	return cloneLobby(s.lobby)
}

func (s *State) lobbySeats() []LobbySeat {
	seats := make([]LobbySeat, len(s.seats))
	for index, seat := range s.seats {
		seats[index] = LobbySeat{Seat: seat.Seat, PlayerNumber: seat.PlayerNumber, Joined: seat.Connected, Locale: seat.Locale}
	}
	return seats
}

func cloneLobby(lobby Lobby) Lobby {
	lobby.Seats = append([]LobbySeat(nil), lobby.Seats...)
	return lobby
}
