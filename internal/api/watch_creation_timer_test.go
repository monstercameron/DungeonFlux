package api

import (
	"testing"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestWatchHub_ReconnectCreationCountdown(t *testing.T) {
	for _, tc := range []struct {
		name             string
		paused, frozen   bool
		total, remaining int64
		delay            time.Duration
		want             int64
	}{
		{"idle elapsed", false, false, 30000, 30000, 7 * time.Second, 23000},
		{"paused", true, false, 30000, 12000, 7 * time.Second, 12000},
		{"frozen", false, true, 30000, 12000, 7 * time.Second, 12000},
		{"disabled", false, false, 0, 0, 7 * time.Second, 0},
		{"overdue", false, false, 30000, 5000, 7 * time.Second, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(time.Unix(0, 0))
			hub := NewWatchHub()
			hub.clock = clk
			timer := domain.TimerView{Name: "creation_timeout", TotalMS: tc.total, RemainingMS: tc.remaining, Frozen: tc.frozen}
			view := domain.View{Path: vocab.StateCreation, Paused: tc.paused, Spotlight: 1, Seats: []domain.SeatView{{Seat: 1, PlayerNumber: 1, TurnTimer: timer}}}
			hub.Publish(view)
			clk.Advance(tc.delay)
			for _, kind := range []df.ClientKind{df.ClientKind_CLIENT_KIND_PHONE, df.ClientKind_CLIENT_KIND_DM} {
				sub := hub.Subscribe(t.Context(), kind, 1)
				state := receiveWatch(t, sub.Messages()).GetState()
				sub.Close()
				got := state.GetPhone().GetTurnTimer()
				if kind == df.ClientKind_CLIENT_KIND_DM {
					got = state.GetDm().GetTurnTimer()
				}
				if got.GetRemainingMs() != tc.want || got.GetFrozen() != tc.frozen {
					t.Fatalf("%v timer = %v, want remaining %d frozen %v", kind, got, tc.want, tc.frozen)
				}
			}
			if hub.latest.Seats[0].TurnTimer != timer {
				t.Fatal("reconnect mutated the cached timer")
			}
		})
	}
}

func TestWatchHub_CreationCountdownRepublishReanchors(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	hub := NewWatchHub()
	hub.clock = clk
	view := domain.View{Path: vocab.StateCreation, Seats: []domain.SeatView{{TurnTimer: domain.TimerView{TotalMS: 30000, RemainingMS: 30000}}}}
	hub.Publish(view)
	clk.Advance(8 * time.Second)
	view.Seats[0].TurnTimer.RemainingMS = 22000
	hub.Publish(view)
	clk.Advance(2 * time.Second)
	for range 2 {
		if got := hub.reconnectView().Seats[0].TurnTimer.RemainingMS; got != 20000 {
			t.Fatalf("remaining = %d, want 20000", got)
		}
	}
}
