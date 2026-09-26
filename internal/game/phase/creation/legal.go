package creation

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	reasonChooseIdentity = "Choose a species and gender first"
	reasonRollHero       = "Roll your hero first"
	reasonReady          = "Ready your rolled hero first"
	reasonLocked         = "Your hero is already ready"
)

// LegalMoveViews returns the complete creation menu for one seat. Moves that
// are not currently accepted remain in the menu with a player-facing reason.
func LegalMoveViews(state SeatState) []domain.MoveView {
	selected := state.Species != "" && state.Gender != ""
	rolled := state.Built
	locked := state.Locked
	return []domain.MoveView{
		creationMove(vocab.MoveSpecies, "Choose species", !rolled && !locked, reasonForCreation(rolled, locked, reasonLocked)),
		creationMove(vocab.MoveGender, "Choose gender", !rolled && !locked, reasonForCreation(rolled, locked, reasonLocked)),
		creationMove(vocab.MoveRollHero, "Roll my hero", selected && !rolled && !locked, rollReason(selected, rolled, locked)),
		creationMove(vocab.MoveReady, "Ready", rolled && !locked, readyReason(rolled, locked)),
	}
}

func creationMove(id vocab.MoveID, label string, enabled bool, reason string) domain.MoveView {
	return domain.MoveView{ID: id, Label: label, Enabled: enabled, Reason: reason}
}

func reasonForCreation(rolled, locked bool, lockedReason string) string {
	if locked {
		return lockedReason
	}
	if rolled {
		return reasonRollHero
	}
	return ""
}

func rollReason(selected, rolled, locked bool) string {
	if locked {
		return reasonLocked
	}
	if rolled {
		return reasonRollHero
	}
	if !selected {
		return reasonChooseIdentity
	}
	return ""
}

func readyReason(rolled, locked bool) string {
	if locked {
		return reasonLocked
	}
	if !rolled {
		return reasonReady
	}
	return ""
}
