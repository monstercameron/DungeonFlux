package main

import "strings"

const phoneSeatStoragePrefix = "dungeonflux.phone.seat_token."

// seatStorageKey returns the localStorage key for one normalized room.
func seatStorageKey(roomCode string) string {
	return phoneSeatStoragePrefix + normalizeRoomCode(roomCode)
}

func hasSavedSeat(roomCode, seatToken string) bool {
	return normalizeRoomCode(roomCode) != "" && strings.TrimSpace(seatToken) != ""
}
