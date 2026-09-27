//go:build js && wasm

package audio

import "syscall/js"

// DebugConsole returns a console logger when the page URL carries
// ?debug=audio, and nil otherwise, so audio reporting costs nothing by default.
func DebugConsole() func(string) {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return nil
	}
	params := js.Global().Get("URLSearchParams").New(location.Get("search"))
	if params.Call("get", "debug").String() != "audio" {
		return nil
	}
	console := js.Global().Get("console")
	return func(line string) { console.Call("log", line) }
}
