// Package dm contains the shared-screen Dungeon Master client.
package dm

import (
	"errors"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/shell/audio"
)

// Seat is the lobby projection for one player position.
type Seat struct {
	Number int
	Name   string
	Ready  bool
	Joined bool
}

// LobbyModel is the data needed to render the DM lobby.
type LobbyModel struct {
	RoomCode     string
	QRURL        string
	OpeningAudio string
	Seats        [2]Seat
	AudioState   string
}

// NewLobbyModel creates a lobby with two empty seats and a join QR URL.
func NewLobbyModel(roomCode, qrURL string) LobbyModel {
	return LobbyModel{
		RoomCode:   roomCode,
		QRURL:      qrURL,
		AudioState: "Waiting for the opening…",
		Seats:      [2]Seat{{Number: 1}, {Number: 2}},
	}
}

// SetSeat replaces one seat's lobby state. Invalid seat numbers are ignored.
func (m *LobbyModel) SetSeat(seat Seat) {
	if m == nil || seat.Number < 1 || seat.Number > len(m.Seats) {
		return
	}
	m.Seats[seat.Number-1] = seat
}

// SeatLabel returns the stable TV label for a seat.
func SeatLabel(seat Seat) string {
	if seat.Name != "" {
		return seat.Name
	}
	return "Player " + string(rune('0'+seat.Number))
}

// ListenAudio schedules Listen frames for a PCM player.
type ListenAudio struct {
	scheduler audio.Scheduler
	player    PCMPlayer
}

// PCMPlayer plays one scheduled PCM chunk in the browser.
type PCMPlayer interface {
	Play(audio.ScheduledChunk) error
}

// NewListenAudio creates a Listen audio adapter.
func NewListenAudio(player PCMPlayer) *ListenAudio {
	return &ListenAudio{player: player}
}

// Handle consumes one Listen message and returns an error for malformed audio
// or when the browser player cannot schedule a chunk.
func (l *ListenAudio) Handle(message *dungeonfluxv1.AudioMessage) error {
	if l == nil || l.player == nil {
		return errors.New("dm lobby: audio player is required")
	}
	if message == nil {
		return errors.New("dm lobby: audio message is required")
	}
	if cancel := message.GetCancel(); cancel != nil {
		if cancel.GetAll() {
			l.scheduler.Cancel("")
		} else {
			l.scheduler.Cancel(cancel.GetUtteranceId())
		}
		return nil
	}
	frame := message.GetFrame()
	chunk, ok, err := l.scheduler.Accept(frame)
	if err != nil {
		return err
	}
	if ok {
		return l.player.Play(chunk)
	}
	return nil
}
