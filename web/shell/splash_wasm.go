//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
	"time"

	"github.com/monstercameron/DungeonFlux/web/dm"
)

// splashCap bounds how long the loading splash can hold the page. A stalled
// asset must never trap the table behind a spinner.
const splashCap = 12 * time.Second

// revealWhenReady lifts the index.html splash (#df-splash) once the mounted
// screen can appear whole, then removes it. The TV waits for its first
// snapshot and its phase's key art (dm.ReadyToReveal); every other client and
// every fixture only waits for the fonts. The screen must report ready on two
// polls in a row so a snapshot racing in mid-check cannot reveal it early.
func revealWhenReady(route Route) {
	ready := func() bool { return true }
	if route == RouteDM && previewName() == "" {
		ready = dm.ReadyToReveal
	}
	go func() {
		start := time.Now()
		streak := 0
		for time.Since(start) < splashCap {
			if fontsLoaded() && ready() {
				streak++
				if streak >= 2 {
					break
				}
			} else {
				streak = 0
			}
			time.Sleep(100 * time.Millisecond)
		}
		if streak < 2 && route == RouteDM && previewName() == "" {
			if console := js.Global().Get("console"); console.Truthy() {
				console.Call("warn", "DungeonFlux splash reached its cap; still waiting for: "+strings.Join(dm.RevealPending(), "; "))
			}
		}
		reveal()
	}()
}

// revealNow lifts the splash immediately, for boot errors that have their
// own screen to show.
func revealNow() { reveal() }

func fontsLoaded() bool {
	fonts := js.Global().Get("document").Get("fonts")
	return !fonts.Truthy() || fonts.Get("status").String() == "loaded"
}

func reveal() {
	doc := js.Global().Get("document")
	if !doc.Truthy() {
		return
	}
	if status := doc.Call("getElementById", "status"); status.Truthy() {
		status.Set("textContent", "Opening the doors")
	}
	root := doc.Get("documentElement").Get("classList")
	root.Call("remove", "df-booting")
	root.Call("add", "df-revealed")
	time.AfterFunc(1200*time.Millisecond, func() {
		if splash := doc.Call("getElementById", "df-splash"); splash.Truthy() {
			splash.Call("remove")
		}
	})
}
