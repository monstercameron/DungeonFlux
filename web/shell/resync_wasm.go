//go:build js && wasm

package main

import "syscall/js"

// resyncRetryMS is the delay before the second resubscribe after a trigger.
const resyncRetryMS = 1500

// installResyncTriggers resubscribes the client's watch streams when the page
// comes back: a phone that slept, switched apps, or changed networks may hold
// a WebSocket that looks open but missed updates. The handlers live for the
// page's lifetime, so they are never released.
func installResyncTriggers(client *Client) {
	if client == nil {
		return
	}
	window := js.Global()
	document := window.Get("document")
	if !document.Truthy() {
		return
	}
	// The "online" event can fire before the network carries traffic, which
	// loses a resubscribe sent at once, so each trigger resubscribes now and
	// again shortly after.
	again := js.FuncOf(func(js.Value, []js.Value) any {
		client.Resync()
		return nil
	})
	resync := js.FuncOf(func(js.Value, []js.Value) any {
		if document.Get("visibilityState").String() != "hidden" {
			client.Resync()
			window.Call("setTimeout", again, resyncRetryMS)
		}
		return nil
	})
	document.Call("addEventListener", "visibilitychange", resync)
	window.Call("addEventListener", "online", resync)
	window.Call("addEventListener", "pageshow", resync)
}
