package content

import "github.com/monstercameron/DungeonFlux/internal/vocab"

// CannedLine is a build-time fallback line and its playback metadata.
type CannedLine struct {
	ID      string
	Role    vocab.Role
	Voice   string
	Text    string
	WordCap int
}

const (
	// CannedOpeningID identifies the opening fallback recording.
	CannedOpeningID = "canned_opening"
	// CannedNPCReplyID identifies the evasive NPC reply recording.
	CannedNPCReplyID = "canned_npc_reply"
	// CannedNPCRevealID identifies the successful persuasion recording.
	CannedNPCRevealID = "canned_npc_reveal"
	// CannedNPCRefuseID identifies the failed persuasion recording.
	CannedNPCRefuseID = "canned_npc_refuse"
	// CannedStrangerFoundID identifies the stranger line when the clue was found.
	CannedStrangerFoundID = "canned_stranger_found"
	// CannedStrangerRelocatedID identifies the relocated-clue stranger line.
	CannedStrangerRelocatedID = "canned_stranger_relocated"
	// CannedCliffhangerNPCID identifies the Mother Vell cliffhanger recording.
	CannedCliffhangerNPCID = "canned_cliffhanger_npc"
	// CannedCliffhangerStrangerID identifies the courier cliffhanger recording.
	CannedCliffhangerStrangerID = "canned_cliffhanger_stranger"
	// CannedSlainBySeat1ID identifies the seat-one combat victory recording.
	CannedSlainBySeat1ID = "canned_slain_by_seat1"
	// CannedSlainBySeat2ID identifies the seat-two combat victory recording.
	CannedSlainBySeat2ID = "canned_slain_by_seat2"
	// CannedFledID identifies the combat flee recording.
	CannedFledID = "canned_fled"
	// CannedExplorationNudgeID identifies the exploration turn nudge.
	CannedExplorationNudgeID = "canned_nudge_exploration"
	// CannedConversationNudgeID identifies the conversation turn nudge.
	CannedConversationNudgeID = "canned_nudge_conversation"
)

var cannedLines = []CannedLine{
	{CannedOpeningID, vocab.RoleOpening, "dm", "Rain hammers the Drowned Lantern. The lamplighter vanished last night, and the river is rising. Two travellers shake off the wet at the bar, where Mother Vell watches with her one good eye.", 40},
	{CannedNPCReplyID, vocab.RoleNPCReply, "mother_vell", "Lots of folk drink here, love. I don't keep a ledger of faces, and I don't answer questions for free.", 25},
	{CannedNPCRevealID, vocab.RoleNPCReveal, "mother_vell", "Fine. They dragged him toward the old bell tower. And at midnight that bell rang, though nobody's climbed it in years.", 25},
	{CannedNPCRefuseID, vocab.RoleNPCRefuse, "mother_vell", "Nice try. I've buried better talkers than you. Drink up or move along; I've nothing more to say.", 25},
	{CannedStrangerFoundID, vocab.RoleStrangerLines, "courier", "A letter, for one of you. The seal's river-soaked, and I didn't read it. Whatever was in that water, it followed me from the river.", 30},
	{CannedStrangerRelocatedID, vocab.RoleStrangerLines, "courier", "A letter, for one of you. They say the lamplighter was dragged to the old bell tower. Something wet and dead guarded it, and it followed me from the river.", 30},
	{CannedCliffhangerNPCID, vocab.RoleCliffhanger, "dm", "Midnight. The tower bell Mother Vell warned of tolls, and every lantern in the tavern gutters out. In the dark it rings again, slow and patient. Whoever pulls that rope already knows your names.", 40},
	{CannedCliffhangerStrangerID, vocab.RoleCliffhanger, "dm", "Midnight. The courier's letter falls open as the tower bell tolls, and every lantern in the tavern gutters out. Inside, in wet ink, are your names. The bell rings again.", 40},
	{CannedSlainBySeat1ID, vocab.RoleCombatOutcomes, "dm", "Steel finds the thrall's heart of river mud, and it collapses into a pool of dark water.", 20},
	{CannedSlainBySeat2ID, vocab.RoleCombatOutcomes, "dm", "One last blow, and the thrall sags. The river takes back its own.", 20},
	{CannedFledID, vocab.RoleCombatOutcomes, "dm", "Far off, the tower bell tolls once. The thrall turns mid-swing and lurches into the rain, toward the tower.", 20},
	{CannedExplorationNudgeID, vocab.RoleOpening, "dm", "The rain won't wait.", 0},
	{CannedConversationNudgeID, vocab.RoleNPCReply, "mother_vell", "Well? Speak or drink.", 0},
}

// CannedLines returns every fixed fallback line in stable spec order.
func CannedLines() []CannedLine {
	lines := make([]CannedLine, len(cannedLines))
	copy(lines, cannedLines)
	return lines
}

// CannedLineByID returns a fallback line by its logical asset ID.
func CannedLineByID(id string) (CannedLine, bool) {
	for _, line := range cannedLines {
		if line.ID == id {
			return line, true
		}
	}
	return CannedLine{}, false
}
