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
	f.ByRole("button", "Español").Click()
	if !strings.Contains(f.Text(), "Unirse a la mesa") {
		t.Fatal("language selection did not update join copy: " + f.Text())
	}
	// Editing another field renders the component again; the explicit choice
	// must still win over the browser's English default.
	f.InputByID("player-name", "Lyria")
	if !strings.Contains(f.Text(), "Unirse a la mesa") {
		t.Fatal("field edit reset the selected language: " + f.Text())
	}
	f.ByRole("button", "English").Click()
	if !strings.Contains(f.Text(), "Join table") {
		t.Fatal("could not switch back to English: " + f.Text())
	}
}
