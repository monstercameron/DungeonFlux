//go:build js && wasm

package main

import (
	"strings"
	"syscall/js"
)

const phoneNameStorageKey = "dungeonflux.phone.player_name"
const legacyPhoneSeatStorageKey = "dungeonflux.phone.seat_token"

func browserSeatToken(roomCode string) string {
	if !hasSavedSeat(roomCode, "saved") {
		return ""
	}
	value := js.Global().Get("localStorage").Call("getItem", seatStorageKey(roomCode))
	if value.Truthy() {
		return strings.TrimSpace(value.String())
	}
	legacy := js.Global().Get("localStorage").Call("getItem", legacyPhoneSeatStorageKey)
	if legacy.IsNull() || legacy.IsUndefined() {
		return ""
	}
	return strings.TrimSpace(legacy.String())
}

func storeBrowserSeatToken(roomCode, token string) {
	if !hasSavedSeat(roomCode, token) {
		return
	}
	js.Global().Get("localStorage").Call("setItem", seatStorageKey(roomCode), strings.TrimSpace(token))
	js.Global().Get("localStorage").Call("removeItem", legacyPhoneSeatStorageKey)
}

func clearBrowserSeatToken(roomCode string) {
	if normalizeRoomCode(roomCode) == "" {
		return
	}
	js.Global().Get("localStorage").Call("removeItem", seatStorageKey(roomCode))
	js.Global().Get("localStorage").Call("removeItem", legacyPhoneSeatStorageKey)
}

func browserPlayerName() string {
	value := js.Global().Get("localStorage").Call("getItem", phoneNameStorageKey)
	if value.IsNull() || value.IsUndefined() {
		return ""
	}
	return value.String()
}

func storeBrowserPlayerName(name string) {
	js.Global().Get("localStorage").Call("setItem", phoneNameStorageKey, strings.TrimSpace(name))
}
