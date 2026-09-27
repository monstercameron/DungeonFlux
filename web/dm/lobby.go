// Package dm contains the shared-screen Dungeon Master client.
package dm

import (
	"errors"
	"net/url"
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/shell/audio"
)

// Seat is the lobby projection for one player position.
type Seat struct {
	Number      int
	Name        string
	Species     string
	Class       string
	Flavor      string
	PortraitURL string
	Ready       bool
	Joined      bool
}

// LobbyModel is the data needed to render the DM lobby.
type LobbyModel struct {
	RoomCode     string
	QRURL        string
	JoinURL      string
	OpeningAudio string
	Seats        [2]Seat
	AudioState   string
	Locale       string
}

// NewLobbyModel creates a lobby with two empty seats and a join QR URL.
func NewLobbyModel(roomCode, qrURL string) LobbyModel {
	roomCode = normalizeRoomCode(roomCode)
	return LobbyModel{
		RoomCode:   roomCode,
		QRURL:      qrURL,
		JoinURL:    JoinURL(roomCode),
		AudioState: T("en", "dm.audio_waiting", nil),
		Seats:      [2]Seat{{Number: 1}, {Number: 2}},
		Locale:     "en",
	}
}

// NewLobbyModelFromDMView converts the live DM snapshot into the lobby model.
// The fallback room code keeps preview and older servers usable while the
// server begins sending the room-level lobby metadata.
func NewLobbyModelFromDMView(view *dungeonfluxv1.DMView, fallbackRoomCode string) LobbyModel {
	model := NewLobbyModel(fallbackRoomCode, "")
	if view == nil {
		return model
	}
	if lobby := view.GetLobby(); lobby != nil {
		if lobby.GetRoomCode() != "" {
			model.RoomCode = normalizeRoomCode(lobby.GetRoomCode())
		}
		if lobby.GetJoinUrl() != "" {
			model.JoinURL = lobby.GetJoinUrl()
		} else {
			model.JoinURL = JoinURL(model.RoomCode)
		}
		if lobby.GetQrUrl() != "" {
			model.QRURL = strings.TrimSpace(lobby.GetQrUrl())
		}
	}
	for index, seat := range view.GetSeats() {
		number := int(seat.GetPlayerNumber())
		if number == 0 {
			number, _ = strconv.Atoi(seat.GetSeatId())
		}
		if number == 0 {
			number = index + 1
		}
		model.SetSeat(Seat{Number: number, Name: seat.GetName(), Joined: seat.GetJoined(), Ready: seat.GetReady()})
	}
	for _, card := range view.GetBuildCards() {
		if card == nil {
			continue
		}
		number := int(card.GetPlayerNumber())
		if number < 1 || number > len(model.Seats) {
			continue
		}
		seat := model.Seats[number-1]
		if card.GetName() != "" {
			seat.Name = card.GetName()
		}
		seat.Class = card.GetClassName()
		seat.PortraitURL = card.GetPortraitUrl()
		seat.Ready = true
		model.Seats[number-1] = seat
	}
	return model
}

func normalizeRoomCode(roomCode string) string {
	return strings.ToUpper(strings.TrimSpace(roomCode))
}

// JoinURL returns the relative phone URL for a room code.
func JoinURL(roomCode string) string {
	roomCode = normalizeRoomCode(roomCode)
	if roomCode == "" {
		return "/p"
	}
	return "/p?" + url.Values{"room": []string{roomCode}}.Encode()
}

// SetLocale settles the render locale for lobby copy.
func (m *LobbyModel) SetLocale(locale string) {
	if m == nil {
		return
	}
	m.Locale = localeOrDefault(locale)
	m.AudioState = T(m.Locale, "dm.audio_waiting", nil)
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
	return SeatName("en", seat.Name, seat.Number)
}

// LobbyStatus returns the concise state shown in the highlighted status plate.
func LobbyStatus(model LobbyModel) string {
	joined := 0
	ready := 0
	for _, seat := range model.Seats {
		if seat.Joined {
			joined++
		}
		if seat.Ready {
			ready++
		}
	}
	if ready == len(model.Seats) {
		return "Begin the tale"
	}
	return "Waiting for players (" + strconv.Itoa(joined) + "/" + strconv.Itoa(len(model.Seats)) + ")"
}

// SeatSubtitle returns the species and class line for one party card.
func SeatSubtitle(seat Seat) string {
	species, class := titleCaseWord(seat.Species), titleCaseWord(seat.Class)
	if species != "" && class != "" {
		return species + " " + class
	}
	if class != "" {
		return class
	}
	if seat.Joined {
		return "Joined"
	}
	return "Adventurer"
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
			if player, ok := l.player.(interface{ Cancel(string) }); ok {
				player.Cancel("")
			}
		} else {
			l.scheduler.Cancel(cancel.GetUtteranceId())
			if player, ok := l.player.(interface{ Cancel(string) }); ok {
				player.Cancel(cancel.GetUtteranceId())
			}
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
