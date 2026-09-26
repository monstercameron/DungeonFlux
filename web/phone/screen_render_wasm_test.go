//go:build js && wasm

package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestRenderPhoneScreen_AllFactories(t *testing.T) {
	props := phoneViewProps{
		creation: NewCreationModel(nil, "seat", 1), sheet: NewSheetModel(),
		moves: NewMovesModel(nil, "seat"), typed: NewTypedInputModel(nil, "seat"),
		dice: NewDiceModel(nil, "seat"), combat: NewCombatModel(nil, "seat"),
		ptt: NewPTTModel(nil, "seat", 1),
	}
	for _, kind := range []ScreenKind{ScreenCreate, ScreenSheet, ScreenMoves, ScreenConversation, ScreenDice, ScreenCombat} {
		t.Run(string(kind), func(t *testing.T) {
			node := renderPhoneScreen(kind, props, "")
			if node == nil {
				t.Fatal("screen factory returned nil")
			}
		})
	}
}

func TestRenderPhoneScreen_SelectsInteractivePhases(t *testing.T) {
	if got := SelectScreen(SeatView{Phase: "conversation"}); got != ScreenConversation {
		t.Fatalf("conversation screen = %q", got)
	}
	if got := SelectScreen(SeatView{Phase: "combat"}); got != ScreenCombat {
		t.Fatalf("combat screen = %q", got)
	}
}

func TestRenderPhoneScreen_RefreshesMovesSnapshot(t *testing.T) {
	model := NewMovesModel(nil, "seat")
	model.state.Moves = []MoveSnapshot{{ID: "talk", Label: "Talk", Enabled: true}}
	first := model.Snapshot()
	model.ApplyScreenState(&df.ScreenState{View: &df.ScreenState_Phone{Phone: &df.PhoneView{Moves: []*df.Move{{MoveId: "leave", Label: "Leave", Enabled: true}}}}})
	second := model.Snapshot()
	if len(first.Moves) != 1 || first.Moves[0].Label != "Talk" || len(second.Moves) != 1 || second.Moves[0].Label != "Leave" {
		t.Fatalf("first=%+v second=%+v", first, second)
	}
}
