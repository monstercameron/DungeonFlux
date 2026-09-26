//go:build js && wasm

package dm

import "github.com/monstercameron/GoWebComponents/v6/ui"

// RenderPreview renders a named fixture through the same DM composition used
// by live Watch updates. Unknown names render the lobby fallback.
func RenderPreview(name, roomCode string, value any) ui.Node {
	fixture, ok := Preview(name)
	if !ok {
		fixture, _ = Preview("lobby")
	}
	unlock, _ := value.(ui.Handler)
	return compose(fixture.State, roomCode, unlock)
}
