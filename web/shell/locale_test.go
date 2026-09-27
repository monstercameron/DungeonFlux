package main

import (
	"context"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestLocaleModel_DetectsAndSwitches(t *testing.T) {
	model := NewLocaleModel([]string{"fr-FR", "es-MX"})
	if model.Active() != "es" {
		t.Fatalf("active = %q, want es", model.Active())
	}
	if got := model.T("shell.join_table", nil); got != "Unirse a la mesa" {
		t.Fatalf("t = %q", got)
	}
	if model.Set("fr") != "es" {
		t.Fatalf("unsupported set changed locale")
	}
	if model.Set("en") != "en" {
		t.Fatalf("set en failed")
	}
	empty := NewLocaleModel(nil)
	if empty.Active() != "en" {
		t.Fatalf("empty tags = %q", empty.Active())
	}
	if got := len(empty.Options()); got != 2 {
		t.Fatalf("options = %d, want 2", got)
	}
}

func TestJoinModel_SendsLocale(t *testing.T) {
	fake := &joinFake{response: &dungeonfluxv1.JoinResponse{SeatId: "1", SeatToken: "token", Locale: "es"}}
	model := NewJoinModel(fake, "room")
	model.SetLocale("es")
	got := model.ApplyJoin(<-model.StartJoin(context.Background(), ""))
	if got.Phase != JoinJoined {
		t.Fatalf("phase = %q", got.Phase)
	}
	if fake.request.GetLocale() != "es" {
		t.Fatalf("request locale = %q", fake.request.GetLocale())
	}
	if got.Locale != "es" {
		t.Fatalf("snapshot locale = %q", got.Locale)
	}
}
