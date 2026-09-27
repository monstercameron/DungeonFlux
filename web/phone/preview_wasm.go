//go:build js && wasm

package phone

import (
	"sync/atomic"
	"syscall/js"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// RenderPreview renders a named fixture through the production phone screen.
// In preview mode the page also exposes window.dfPreview(name), which swaps the
// fixture in place so screen-to-screen transitions can be reviewed.
func RenderPreview(name string) (ui.Node, bool) {
	if _, ok := Preview(name); !ok {
		return nil, false
	}
	return ui.CreateElement(previewScreen, previewScreenProps{name: name, revision: previewRevision.Add(1)}), true
}

type previewScreenProps struct {
	name     string
	revision uint64
}

var previewRevision atomic.Uint64

// previewPick is the fixture chosen through window.dfPreview; it survives a
// shell re-render (art loaded).
var previewPick string

func previewScreen(props previewScreenProps) ui.Node {
	picks := ui.UseState(0)
	ui.UseEffect(func() func() {
		pick := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 && args[0].Type() == js.TypeString {
				if _, ok := Preview(args[0].String()); ok {
					previewPick = args[0].String()
					picks.Set(picks.Get() + 1)
				}
			}
			return nil
		})
		js.Global().Set("dfPreview", pick)
		return func() {
			js.Global().Delete("dfPreview")
			pick.Release()
		}
	}, props.name)
	name := props.name
	if previewPick != "" {
		name = previewPick
	}
	fixture, _ := Preview(name)
	viewProps := previewProps(fixture.View)
	viewProps.journal = NewJournalLog()
	return renderPhoneScreen(SelectScreen(fixture.View), viewProps, fixture.View.Phone.GetLocale(), fixture.View, PhoneTabPlay, nil, ui.Handler{})
}

func previewProps(view SeatView) phoneViewProps {
	props := phoneViewProps{
		creation: NewCreationModel(nil, "preview-seat", 1), sheet: NewSheetModel(),
		moves: NewMovesModel(nil, "preview-seat"), typed: NewTypedInputModel(nil, "preview-seat"),
		dice: NewDiceModel(nil, "preview-seat"), combat: NewCombatModel(nil, "preview-seat"),
		ptt: previewPTT(view), end: NewEndModel(),
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
