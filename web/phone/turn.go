package phone

import (
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// TurnStatus is the persistent, phase-agnostic label that tells a player
// whether the floor is theirs: which seat currently has the spotlight (or,
// in combat, which combatant has the initiative), and whether that is this
// phone's own seat.
type TurnStatus struct {
	// Yours is true when this phone's seat currently holds the spotlight
	// or, in combat, the acting token belongs to this seat.
	Yours bool
	// Known is false when the engine has not assigned a turn yet (for
	// example the lobby, character creation, or the end card), so no
	// banner should render.
	Known bool
	// ActorName is the other actor's display name, used when Yours is
	// false. It is empty when the name could not be resolved, and the
	// caller falls back to a generic label.
	ActorName string
}

// turnPhases lists the phases where a turn or spotlight banner is shown.
// Character creation, the lobby, and the end card have their own framing
// and never show a turn label.
func turnRelevant(phase string) bool {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "exploration", "conversation", "check", "combat", "opening", "resolution", "hook_event", "cliffhanger":
		return true
	default:
		return false
	}
}

// ComputeTurnStatus derives the current turn label from a seat view. Combat
// uses the token turn order (which includes the thrall by name); every other
// phase uses the engine's spotlight seat compared against the lobby roster.
func ComputeTurnStatus(view SeatView) TurnStatus {
	if !turnRelevant(view.Phase) {
		return TurnStatus{}
	}
	if combat := view.Phone.GetCombat(); combat != nil {
		return combatTurnStatus(combat, view.Phone.GetTurnOrder())
	}
	return spotlightTurnStatus(view)
}

func combatTurnStatus(combat *df.CombatView, turnOrder []*df.TurnOrderEntry) TurnStatus {
	if combat.GetMyTurn() {
		return TurnStatus{Yours: true, Known: true}
	}
	for _, entry := range turnOrder {
		if entry.GetActive() {
			return TurnStatus{Known: true, ActorName: strings.TrimSpace(entry.GetName())}
		}
	}
	return TurnStatus{Known: true}
}

func spotlightTurnStatus(view SeatView) TurnStatus {
	spotlight := strings.TrimSpace(view.SpotlightSeat)
	if spotlight == "" {
		return TurnStatus{}
	}
	spotlightNumber, err := strconv.Atoi(spotlight)
	if err != nil {
		return TurnStatus{}
	}
	if int32(spotlightNumber) == view.PlayerNumber && view.PlayerNumber > 0 {
		return TurnStatus{Yours: true, Known: true}
	}
	for _, seat := range view.LobbySeats {
		if seat != nil && seat.GetPlayerNumber() == int32(spotlightNumber) {
			return TurnStatus{Known: true, ActorName: strings.TrimSpace(seat.GetName())}
		}
	}
	return TurnStatus{Known: true}
}

// TurnBannerText renders the localized banner text for a turn status.
func TurnBannerText(locale string, status TurnStatus) string {
	if !status.Known {
		return ""
	}
	if status.Yours {
		return T(locale, "turn.yours", nil)
	}
	if status.ActorName != "" {
		return T(locale, "turn.waiting_named", map[string]string{"name": status.ActorName})
	}
	return T(locale, "turn.waiting", nil)
}
