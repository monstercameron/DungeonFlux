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

// SelectScreen maps every demo phase to the phone screen that owns its actions.
func SelectScreen(view SeatView) ScreenKind {
	phase := strings.ToLower(strings.TrimSpace(view.Phase))
	switch {
	case strings.Contains(phase, "lobby"), strings.Contains(phase, "exploration"):
		return ScreenMoves
	case strings.Contains(phase, "creation"):
		return ScreenCreate
	case strings.Contains(phase, "check"):
		return ScreenDice
	case strings.Contains(phase, "combat"):
		return ScreenCombat
	case strings.Contains(phase, "conversation"):
		return ScreenConversation
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
