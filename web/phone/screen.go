package phone

import (
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ScreenKind identifies the phone component selected for a server phase.
type ScreenKind string

const (
	// ScreenWaiting shows the player's seat while the host gathers the table.
	ScreenWaiting ScreenKind = "waiting"
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
	// ScreenEnd shows the final outcome and character state.
	ScreenEnd ScreenKind = "end"
)

// SeatView is the seat-specific state used to select and render a phone screen.
type SeatView struct {
	Version      uint64
	Phase        string
	Phone        *df.PhoneView
	Narration    NarrationModel
	PlayerName   string
	PlayerNumber int32
	LobbySeats   []*df.LobbySeat
	// SpotlightSeat is the seat number (as a string) the engine currently
	// gives the floor to, in every phase, not just combat.
	SpotlightSeat string
}

// NarrationModel is the phone-safe read-along line projection.
type NarrationModel struct {
	Speaker string
	Text    string
	Done    bool
}

func seatViewFromState(state *df.ScreenState) SeatView {
	if state == nil {
		return SeatView{}
	}
	phone := state.GetPhone()
	model := NarrationModel{}
	if narration := phone.GetNarration(); narration != nil {
		model = NarrationModel{Speaker: narration.GetSpeaker(), Text: narration.GetTextSoFar(), Done: narration.GetDone()}
	}
	return SeatView{
		Version: state.GetVersion(), Phase: state.GetPhase(), Phone: phone, Narration: model,
		PlayerName: phone.GetPlayerName(), PlayerNumber: phone.GetPlayerNumber(), LobbySeats: phone.GetSeats(),
		SpotlightSeat: state.GetSpotlightSeat(),
	}
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
	Location    string
	Act         string
	Scene       string
	Locale      string
	Connection  ConnectionState
	Screen      ScreenKind
	Mode        PhoneMode
	ActiveTab   PhoneTabID
	// TurnLabel is the persistent whose-turn banner text ("Your turn",
	// "Waiting for Lyra…"), or empty when no turn is relevant on this
	// screen (lobby, creation, end).
	TurnLabel string
	// TurnYours highlights TurnLabel gold when it is this seat's turn.
	TurnYours bool
	// Enter plays the short screen entry (fade and 8 px rise) on this frame.
	// The mount sets it only on the frame where the screen kind changed.
	Enter bool
}

// NewFrameModel creates a frame with sensible labels for a seat.
func NewFrameModel(displayName, locale string) FrameModel {
	if displayName == "" {
		displayName = "Player"
	}
	if locale == "" {
		locale = "en"
	}
	return FrameModel{
		DisplayName: displayName, Title: "Your adventure", Location: "The Drowned Lantern",
		Act: "Act I", Scene: "Scene 1", Locale: locale, Connection: ConnectionConnecting,
		Mode: PhoneModePlay, ActiveTab: PhoneTabPlay,
	}
}

// ApplyView updates the frame screen and marks the connection online.
func (m *FrameModel) ApplyView(view SeatView) ScreenTransition {
	if m == nil {
		return ScreenTransition{}
	}
	previous := m.Screen
	m.Screen = SelectScreen(view)
	if m.Screen == ScreenCombat {
		m.Location = T(m.Locale, "combat.location", nil)
	} else {
		m.Location = "The Drowned Lantern"
	}
	m.Mode = modeForScreen(m.Screen)
	m.ActiveTab = PhoneTabPlay
	m.Connection = ConnectionOnline
	return ScreenTransition{From: previous, To: m.Screen, Changed: previous != m.Screen}
}

func modeForScreen(screen ScreenKind) PhoneMode {
	switch screen {
	case ScreenCombat:
		return PhoneModeCombat
	case ScreenMoves:
		return PhoneModeExplore
	default:
		return PhoneModePlay
	}
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
	case "lobby":
		if view.PlayerName != "" || view.PlayerNumber > 0 || len(view.LobbySeats) > 0 {
			return ScreenWaiting
		}
		return ScreenMoves
	case "exploration":
		return ScreenMoves
	case "creation":
		return ScreenCreate
	case "check":
		return ScreenDice
	case "combat":
		return ScreenCombat
	case "conversation":
		return ScreenConversation
	case "resolution":
		// INT-009/PHONE-033: resolution is the outcome beat for the check
		// that just ran (the d20 face, total, and Success/Failure banner),
		// so it stays on the dice screen instead of the generic sheet.
		return ScreenDice
	case "opening", "hook_event", "cliffhanger":
		return ScreenSheet
	case "end":
		return ScreenEnd
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
