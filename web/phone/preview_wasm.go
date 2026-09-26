//go:build js && wasm

package phone

import (
	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// RenderPreview renders a named fixture through the production phone screen.
func RenderPreview(name string) (ui.Node, bool) {
	fixture, ok := Preview(name)
	if !ok {
		return nil, false
	}
	props := previewProps(fixture.View)
	return renderPhoneScreen(SelectScreen(fixture.View), props, fixture.View.Phone.GetLocale()), true
}

func previewProps(view SeatView) phoneViewProps {
	props := phoneViewProps{
		creation: NewCreationModel(nil, "preview-seat", 1), sheet: NewSheetModel(),
		moves: NewMovesModel(nil, "preview-seat"), typed: NewTypedInputModel(nil, "preview-seat"),
		dice: NewDiceModel(nil, "preview-seat"), combat: NewCombatModel(nil, "preview-seat"),
		ptt: NewPTTModel(nil, "preview-seat", 1), end: NewEndModel(),
	}
	state := &df.ScreenState{Phase: view.Phase, View: &df.ScreenState_Phone{Phone: view.Phone}}
	props.creation.ApplyScreenState(state)
	props.sheet.ApplyScreenState(state)
	props.moves.ApplyScreenState(state)
	props.dice.ApplyScreenState(state)
	props.combat.ApplyScreenState(state)
	props.end.ApplyScreenState(state)
	return props
}
