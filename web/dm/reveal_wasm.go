//go:build js && wasm

package dm

import (
	"strconv"
	"sync"
	"sync/atomic"
	"syscall/js"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// revealScreen is the newest live snapshot the TV has rendered. Fixtures do
// not set it: a preview has no server and reveals as soon as it mounts.
var revealScreen atomic.Pointer[dungeonfluxv1.ScreenState]

func noteRevealScreen(state *dungeonfluxv1.ScreenState) {
	if state != nil {
		revealScreen.Store(state)
	}
}

// ReadyToReveal reports whether the TV can fade in whole: the first snapshot
// has arrived and every piece of art its phase leads with (revealArt) is
// downloaded and decoded. The shell polls it while the loading splash is up.
func ReadyToReveal() bool { return len(RevealPending()) == 0 }

// RevealPending lists what the TV is still waiting for, for the shell's
// console warning when the splash hits its time cap.
func RevealPending() []string {
	state := revealScreen.Load()
	if state == nil {
		return []string{"first snapshot"}
	}
	view := state.GetDm()
	qr := NewLobbyModelFromDMView(view, dmRoomCode()).QRURL
	urls, _ := revealArt(state.GetPhase(), currentAspectClass(), qr, sceneBackgroundURL(view), ArtURL)
	var pending []string
	for i, url := range urls {
		switch {
		case url == "":
			pending = append(pending, "art #"+strconv.Itoa(i)+" not downloaded (phase "+state.GetPhase()+", qr "+qr+")")
		case !decoded(url): // keep going so every decode starts in parallel
			pending = append(pending, "decoding "+url)
		}
	}
	return pending
}

var (
	decodeMu    sync.Mutex
	decodeState = map[string]int{} // 1 decoding, 2 done (or failed; a broken image must not hold the splash)
)

// decoded starts decoding url once and reports whether it has finished, so
// the first frame after the fade already has the bitmap and nothing pops.
func decoded(url string) bool {
	decodeMu.Lock()
	defer decodeMu.Unlock()
	switch decodeState[url] {
	case 2:
		return true
	case 1:
		return false
	}
	decodeState[url] = 1
	img := js.Global().Get("Image").New()
	var done js.Func
	done = js.FuncOf(func(js.Value, []js.Value) any {
		decodeMu.Lock()
		decodeState[url] = 2
		decodeMu.Unlock()
		done.Release()
		return nil
	})
	img.Set("src", url)
	img.Call("decode").Call("then", done, done)
	return false
}
