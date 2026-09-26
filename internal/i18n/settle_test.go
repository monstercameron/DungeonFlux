package i18n

import (
	"testing"
)

func TestSeatLocales_Table(t *testing.T) {
	seats := NewSeatLocales("es")
	if got := seats.Room(); got != "es" {
		t.Fatalf("Room() = %q, want es", got)
	}
	if got := seats.For("p1"); got != "es" {
		t.Fatalf("For(p1) before join = %q, want room default es", got)
	}
	if got := seats.Settle("p1", "en"); got != "en" {
		t.Fatalf("Settle(p1, en) = %q, want en", got)
	}
	if got := seats.For("p1"); got != "en" {
		t.Fatalf("For(p1) after join = %q, want en", got)
	}
	if got := seats.For("p2"); got != "es" {
		t.Fatalf("For(p2) = %q, want room default es", got)
	}
	if got := seats.SetRoom("en"); got != "en" {
		t.Fatalf("SetRoom(en) = %q, want en", got)
	}
	if got := seats.Room(); got != "en" {
		t.Fatalf("Room() = %q, want en", got)
	}
	if got := seats.For("p2"); got != "en" {
		t.Fatalf("For(p2) after room change = %q, want en", got)
	}

	var nilTable *SeatLocales
	if got := nilTable.Settle("p1", "es"); got != "es" {
		t.Fatalf("nil Settle = %q, want es", got)
	}
	if got := nilTable.For("p1"); got != DefaultLocale {
		t.Fatalf("nil For = %q, want %q", got, DefaultLocale)
	}
	if got := nilTable.Room(); got != DefaultLocale {
		t.Fatalf("nil Room = %q, want %q", got, DefaultLocale)
	}
	if got := nilTable.SetRoom("es"); got != "es" {
		t.Fatalf("nil SetRoom = %q, want es", got)
	}
}
