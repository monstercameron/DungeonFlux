package phone

import "strings"

// TalkSnapshot is the render-safe conversation copy used by the phone.
// The server remains authoritative for the legal move list; this type only
// supplies the NPC presentation that is not part of PhoneView yet.
type TalkSnapshot struct {
	Speaker     string
	Role        string
	Quote       string
	PortraitArt string
}

// NewTalkSnapshot projects the current status into a stable NPC presentation.
func NewTalkSnapshot(status string) TalkSnapshot {
	quote := "You are not the first to ask about the lamplighter tonight. And you will not be the last."
	if strings.TrimSpace(status) != "" && !isPTTStatus(status) {
		quote = strings.TrimSpace(status)
	}
	return TalkSnapshot{
		Speaker:     "Marra",
		Role:        "Barkeep",
		Quote:       quote,
		PortraitArt: "mother_vell",
	}
}

// TalkMoveRows converts legal conversation moves to shared choice-row models.
func TalkMoveRows(moves []MoveSnapshot) []ChoiceRowModel {
	rows := make([]ChoiceRowModel, 0, len(moves))
	highlighted := false
	for _, move := range moves {
		row := ChoiceRowModel{
			ID:          move.ID,
			Label:       move.Label,
			Reason:      move.Reason,
			Icon:        moveArtAsset(move.ID),
			Enabled:     move.Enabled,
			Highlighted: move.Enabled && !highlighted,
		}
		if row.Highlighted {
			highlighted = true
		}
		rows = append(rows, row)
	}
	return rows
}

func isPTTStatus(status string) bool {
	lower := strings.ToLower(status)
	for _, word := range []string{"speak", "listening", "transcrib", "speech", "type your"} {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}
