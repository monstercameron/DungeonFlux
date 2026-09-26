//go:build js && wasm

package main

import "syscall/js"

// rememberBootQuery copies the access tokens from the first URL into
// sessionStorage before the history router rewrites the address (it drops the
// query string when it normalises /dm), so the DM and host screens can still
// authenticate their streams.
func rememberBootQuery() {
	location := js.Global().Get("location")
	storage := js.Global().Get("sessionStorage")
	if !location.Truthy() || !storage.Truthy() {
		return
	}
	params := js.Global().Get("URLSearchParams").New(location.Get("search"))
	for _, pair := range [][2]string{{"token", "df-dm-token"}, {"t", "df-host-token"}, {"room", "df-room"}} {
		value := params.Call("get", pair[0])
		if value.Truthy() && value.String() != "" {
			storage.Call("setItem", pair[1], value.String())
		}
	}
}
