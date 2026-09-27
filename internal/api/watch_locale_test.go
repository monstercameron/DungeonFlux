package api

import (
	"context"
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestWatchHub_UsesJoinedLocaleForInitialAndReconnectedPhones(t *testing.T) {
	hub := NewWatchHub()
	view := testLocaleView()
	view.Seats[0].Locale = "es"
	view.Seats = append(view.Seats, domain.SeatView{Seat: 2, Locale: "en"})
	hub.Publish(view)
	for range 2 {
		ctx, cancel := context.WithCancel(context.Background())
		sub := hub.Subscribe(ctx, df.ClientKind_CLIENT_KIND_PHONE, 1)
		state := receiveWatch(t, sub.Messages()).GetState()
		sub.Close()
		cancel()
		if state.GetPhone().GetLocale() != "es" || state.GetPhone().GetMoves()[0].GetLabel() != "Atacar al ahogado" {
			t.Fatalf("joined locale ignored: %v", state)
		}
		seats := state.GetPhone().GetSeats()
		if seats[0].GetLocale() != "es" || seats[1].GetLocale() != "en" {
			t.Fatalf("viewer language overwrote party identities: %v", seats)
		}
	}
	if view.Seats[0].Locale != "es" || view.Seats[1].Locale != "en" {
		t.Fatal("projection mutated source")
	}
}

func TestWatchHub_LocaleFallbackAndExplicitOverride(t *testing.T) {
	for _, tc := range []struct{ name, room, seat, override, want string }{
		{"default", "", "", "", "en"},
		{"room", "es", "", "", "es"},
		{"seat", "en", "es", "", "es"},
		{"override", "es", "es", "en", "en"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hub := NewWatchHub()
			if tc.override != "" {
				hub.RememberLocale(1, tc.override)
			}
			view := domain.View{Locale: tc.room, Seats: []domain.SeatView{{Seat: 1, Locale: tc.seat}}}
			if got := hub.localeFor(1, view); got != tc.want {
				t.Fatalf("locale = %q, want %q", got, tc.want)
			}
		})
	}
}
