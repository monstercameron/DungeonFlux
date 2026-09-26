package phone

import (
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ScreenKind identifies the phone component selected for a server phase.
type ScreenKind string

const (
	// ScreenCreate shows species, gender, and hero rolling.
	ScreenCreate ScreenKind = "create"
	// ScreenSheet shows the current character and status.
	ScreenSheet ScreenKind = "sheet"
	// ScreenMoves shows the server-authoritative legal moves.
	ScreenMoves ScreenKind = "moves"
	// ScreenConversation shows moves and the typed speech fallback.
	ScreenConversation ScreenKind = "conversation"
	// ScreenDice shows the persuasion check.
	ScreenDice ScreenKind = "dice"
	// ScreenCombat shows movement and attack controls.
	ScreenCombat ScreenKind = "combat"
)

// SeatView is the seat-specific state used to select and render a phone screen.
type SeatView struct {
	Phase string
	Phone *df.PhoneView
}

// ConnectionState identifies the transport state shown in the phone header.
type ConnectionState string

const (
	// ConnectionOnline means the watch stream is receiving current state.
	ConnectionOnline ConnectionState = "online"
	// ConnectionConnecting means the client is opening or restoring its stream.
	ConnectionConnecting ConnectionState = "connecting"
	// ConnectionOffline means actions cannot currently reach the server.
	ConnectionOffline ConnectionState = "offline"
)

// FrameModel is the small, render-safe state shared by the phone frame.
type FrameModel struct {
	DisplayName string
	Title       string
	Locale      string
	Connection  ConnectionState
	Screen      ScreenKind
}

// NewFrameModel creates a frame with sensible labels for a seat.
func NewFrameModel(displayName, locale string) FrameModel {
	if displayName == "" {
		displayName = "Player"
	}
	if locale == "" {
		locale = "en"
	}
	return FrameModel{DisplayName: displayName, Title: "Your adventure", Locale: locale, Connection: ConnectionConnecting}
}

// ApplyView updates the frame screen and marks the connection online.
func (m *FrameModel) ApplyView(view SeatView) ScreenTransition {
	if m == nil {
		return ScreenTransition{}
	}
	previous := m.Screen
	m.Screen = SelectScreen(view)
	m.Connection = ConnectionOnline
	return ScreenTransition{From: previous, To: m.Screen, Changed: previous != m.Screen}
}

// SetConnection records a transport state for the next render.
func (m *FrameModel) SetConnection(state ConnectionState) {
	if m != nil {
		m.Connection = state
	}
}

// ScreenTransition describes a phase-driven phone screen change.
type ScreenTransition struct {
	From    ScreenKind
	To      ScreenKind
	Changed bool
}

// SelectScreen maps every demo phase to the phone screen that owns its actions.
func SelectScreen(view SeatView) ScreenKind {
	phase := strings.ToLower(strings.TrimSpace(view.Phase))
	switch phase {
	case "lobby", "exploration":
		return ScreenMoves
	case "creation":
		return ScreenCreate
	case "check":
		return ScreenDice
	case "combat":
		return ScreenCombat
	case "conversation":
		return ScreenConversation
	case "opening", "resolution", "hook_event", "cliffhanger", "end":
		return ScreenSheet
	default:
		if view.Phone != nil {
			if view.Phone.GetCombat() != nil {
				return ScreenCombat
			}
			for _, move := range view.Phone.GetMoves() {
				if move.GetMoveId() == "persuade" {
					return ScreenDice
				}
			}
			if len(view.Phone.GetMoves()) > 0 {
				return ScreenMoves
			}
			if view.Phone.GetCharacter() == nil {
				return ScreenCreate
			}
		}
		return ScreenSheet
	}
}
