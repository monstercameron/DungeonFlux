//go:build js && wasm

package dm

import (
	"strconv"
	"sync"
	"syscall/js"
)

// canvasScaleOnce guards the single resize listener for the whole page.
var canvasScaleOnce sync.Once

// installCanvasScale keeps the --df-scale CSS variable equal to the factor that
// fits the fixed 1920x1080 DM canvas inside the viewport. CSS cannot divide a
// viewport length by a pixel length portably, so the factor is computed here.
func installCanvasScale() {
	canvasScaleOnce.Do(func() {
		window := js.Global().Get("window")
		update := js.FuncOf(func(js.Value, []js.Value) any {
			width := window.Get("innerWidth").Float()
			height := window.Get("innerHeight").Float()
			js.Global().Get("document").Get("documentElement").Get("style").Call(
				"setProperty", "--df-scale", strconv.FormatFloat(CanvasScale(width, height), 'f', 5, 64))
			return nil
		})
		window.Call("addEventListener", "resize", update)
		update.Invoke()
	})
}
