package main

import "testing"

func TestSeatStorageKey_IsRoomScoped(t *testing.T) {
	if got := seatStorageKey(" ab12 "); got != "dungeonflux.phone.seat_token.AB12" {
		t.Fatalf("key = %q", got)
	}
	if seatStorageKey("AB12") == seatStorageKey("CD34") {
		t.Fatal("different rooms share a seat key")
	}
}

func TestHasSavedSeat_RejectsIncompleteIdentity(t *testing.T) {
	if !hasSavedSeat("room", "token") {
		t.Fatal("valid room and token rejected")
	}
	if hasSavedSeat("", "token") || hasSavedSeat("room", " ") {
		t.Fatal("incomplete saved seat accepted")
	}
}
