//go:build js && wasm

package dm

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// RenderPreview renders a named fixture through the same DM composition used
// by live Watch updates. Unknown names render the lobby fallback.
//
// In preview mode the page also exposes window.dfPreview(name), which swaps
// the fixture in place so phase transitions can be reviewed without playing
// (for example dfPreview("creation") on /dm?preview=lobby).
func RenderPreview(name, roomCode string, value any) ui.Node {
	unlock, _ := value.(ui.Handler)
	return ui.CreateElement(previewScreen, previewProps{name: name, roomCode: roomCode, unlock: unlock})
}

type previewProps struct {
	name     string
	roomCode string
	unlock   ui.Handler
}

// previewPick is the fixture chosen through window.dfPreview. It lives outside
// the component so a shell re-render (art loaded) keeps the chosen fixture.
var previewPick string

func previewScreen(props previewProps) ui.Node {
	picks := ui.UseState(0)
	ui.UseEffect(func() func() {
		pick := js.FuncOf(func(_ js.Value, args []js.Value) any {
			if len(args) > 0 && args[0].Type() == js.TypeString {
				previewPick = args[0].String()
				picks.Set(picks.Get() + 1)
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
	fixture, ok := Preview(name)
	if !ok {
		fixture, _ = Preview("lobby")
	}
	useTransitionClock(fixture.State.GetPhase())
	return compose(fixture.State, props.roomCode, props.unlock)
}
