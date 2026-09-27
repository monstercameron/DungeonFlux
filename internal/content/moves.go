// Package content contains the authored data used by the DungeonFlux engine
// and clients.
package content

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ReasonCode identifies an engine condition that disables a move in the
// phone's action list.
type ReasonCode string

const (
	ReasonWaitingForPlayer    ReasonCode = "WAITING_FOR_PLAYER"
	ReasonConversationDone    ReasonCode = "CONVERSATION_DONE"
	ReasonConversationRefused ReasonCode = "CONVERSATION_REFUSED"
	ReasonTalkFirst           ReasonCode = "TALK_FIRST"
	ReasonSpeaking            ReasonCode = "SPEAKING"
	ReasonNoPath              ReasonCode = "NO_PATH"
	ReasonNoMovement          ReasonCode = "NO_MOVEMENT"
	ReasonDown                ReasonCode = "DOWN"
	ReasonBuildingHero        ReasonCode = "BUILDING_HERO"
	ReasonMissingChoice       ReasonCode = "MISSING_CHOICE"
	ReasonNotYourTurn         ReasonCode = "NOT_YOUR_TURN"
	ReasonNoSlot              ReasonCode = "NO_SLOT"
	ReasonOutOfRange          ReasonCode = "OUT_OF_RANGE"
	ReasonIncapacitated       ReasonCode = "INCAPACITATED"
)

// MoveLabel returns the phone label for a move. It returns an empty string
// for a move outside the closed vocabulary.
func MoveLabel(id vocab.MoveID) string {
	switch id {
	case vocab.MoveReady:
		return "Ready"
	case vocab.MoveSpecies:
		return "Choose a species"
	case vocab.MoveGender:
		return "Choose a gender"
	case vocab.MoveRollHero:
		return "Roll my hero"
	case vocab.MoveTalkVell:
		return "Talk to Mother Vell"
	case vocab.MovePersuade:
		return "Persuade +4 vs DC 10"
	case vocab.MoveStepAway:
		return "Step away"
	case vocab.MoveLeave:
		return "Leave"
	case vocab.MoveAttack:
		return "Attack the drowned thrall"
	case vocab.MoveMove:
		return "Move"
	case vocab.MoveEndTurn:
		return "End turn"
	default:
		return ""
	}
}

// MoveLabels returns a fresh catalog containing a label for every MoveID in
// vocab. Callers may modify the returned map without changing this package.
func MoveLabels() map[vocab.MoveID]string {
	labels := make(map[vocab.MoveID]string, 11)
	for _, id := range []vocab.MoveID{
		vocab.MoveReady, vocab.MoveSpecies, vocab.MoveGender, vocab.MoveRollHero,
		vocab.MoveTalkVell, vocab.MovePersuade, vocab.MoveStepAway, vocab.MoveLeave,
		vocab.MoveAttack, vocab.MoveMove, vocab.MoveEndTurn,
	} {
		labels[id] = MoveLabel(id)
	}
	return labels
}

// ReasonText returns localized phone copy for a rejection reason. The name
// parameter is used by waiting-for-player and holder by not-your-turn; ft,
// level, and name are ignored when a reason does not need them.
func ReasonText(code ReasonCode, params map[string]string) string {
	value := func(key string) string {
		if params == nil {
			return ""
		}
		return params[key]
	}

	switch code {
	case ReasonWaitingForPlayer:
		return fmt.Sprintf("Waiting for %s", value("name"))
	case ReasonConversationDone:
		return "She's told you what she knows"
	case ReasonConversationRefused:
		return "She won't say more"
	case ReasonTalkFirst:
		return "Talk to her first"
	case ReasonSpeaking:
		return "Mother Vell is speaking"
	case ReasonNoPath:
		return "No path to the thrall"
	case ReasonNoMovement:
		return "No movement left"
	case ReasonDown:
		return "You're down, the others fight on"
	case ReasonBuildingHero:
		return "Building your hero…"
	case ReasonMissingChoice:
		return "Choose a species and gender first"
	case ReasonNotYourTurn:
		return fmt.Sprintf("Waiting for %s", value("holder"))
	case ReasonNoSlot:
		return fmt.Sprintf("No %s-level slots left", value("level"))
	case ReasonOutOfRange:
		return fmt.Sprintf("Target out of range (%s ft)", value("ft"))
	case ReasonIncapacitated:
		return "You can't act while incapacitated"
	default:
		return ""
	}
}

// RejectionReasons returns every authored rejection reason and its canonical
// text. Parameterized reasons use their documented placeholder form.
func RejectionReasons() map[ReasonCode]string {
	return map[ReasonCode]string{
		ReasonWaitingForPlayer:    "Waiting for {name}",
		ReasonConversationDone:    ReasonText(ReasonConversationDone, nil),
		ReasonConversationRefused: ReasonText(ReasonConversationRefused, nil),
		ReasonTalkFirst:           ReasonText(ReasonTalkFirst, nil),
		ReasonSpeaking:            ReasonText(ReasonSpeaking, nil),
		ReasonNoPath:              ReasonText(ReasonNoPath, nil),
		ReasonNoMovement:          ReasonText(ReasonNoMovement, nil),
		ReasonDown:                ReasonText(ReasonDown, nil),
		ReasonBuildingHero:        ReasonText(ReasonBuildingHero, nil),
		ReasonMissingChoice:       ReasonText(ReasonMissingChoice, nil),
		ReasonNotYourTurn:         "Waiting for {holder}",
		ReasonNoSlot:              "No {level}-level slots left",
		ReasonOutOfRange:          "Target out of range ({ft} ft)",
		ReasonIncapacitated:       ReasonText(ReasonIncapacitated, nil),
	}
}
