//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/testkit/render"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"strings"
	"syscall/js"
	"testing"
)

func TestJoinRender_NameThenRoomUpdatesFrame(t *testing.T) {
	f := render.New(t)
	js.Global().Set("location", js.ValueOf(map[string]any{"search": ""}))
	js.Global().Set("navigator", js.ValueOf(map[string]any{"languages": []any{"en"}, "language": "en"}))
	getItem := js.FuncOf(func(js.Value, []js.Value) any { return nil })
	defer getItem.Release()
	js.Global().Set("localStorage", js.ValueOf(map[string]any{"getItem": getItem}))
	f.Render(ui.CreateElement(JoinScreen(nil)))
	f.InputByID("player-name", "Lyra")
	f.InputByID("room-code", "DF-FAKE")
	if !strings.Contains(f.Text(), "Room link recognized") {
		t.Fatal("room entry stayed stale: " + f.Text())
	}
}
