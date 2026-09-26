package phone

import "testing"

func TestNewTalkSnapshot_UsesNPCDefaultsAndLiveStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		quote  string
	}{
		{name: "default", status: "", quote: "You are not the first to ask about the lamplighter tonight. And you will not be the last."},
		{name: "ptt status keeps npc line", status: "Press and hold to speak", quote: "You are not the first to ask about the lamplighter tonight. And you will not be the last."},
		{name: "server line", status: "The room falls silent.", quote: "The room falls silent."},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewTalkSnapshot(test.status)
			if got.Speaker != "Marra" || got.Role != "Barkeep" || got.PortraitArt != "mother_vell" || got.Quote != test.quote {
				t.Fatalf("snapshot = %+v", got)
			}
		})
	}
}

func TestTalkMoveRows_HighlightsFirstEnabledAndKeepsReasons(t *testing.T) {
	rows := TalkMoveRows([]MoveSnapshot{
		{ID: "persuade", Label: "Persuade", Enabled: false, Reason: "Wait until she stops speaking"},
		{ID: "talk_vell", Label: "Ask what she knows", Enabled: true},
		{ID: "leave", Label: "Step away", Enabled: true},
	})
	if len(rows) != 3 || rows[0].Icon != "ui/icon_persuade" || rows[0].Reason == "" || !rows[1].Highlighted || rows[2].Highlighted {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestTalkMoveRows_DoesNotHighlightDisabledMoves(t *testing.T) {
	rows := TalkMoveRows([]MoveSnapshot{{ID: "persuade", Enabled: false}, {ID: "leave", Enabled: false}})
	for _, row := range rows {
		if row.Highlighted {
			t.Fatalf("disabled row highlighted: %+v", row)
		}
	}
}
