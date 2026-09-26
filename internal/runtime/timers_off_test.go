package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestTimers_TurnTimersOffCancelsPacingButKeepsIdleAndCombatCap(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox(4)
	timers := NewTimers(clk, inbox)
	for _, timer := range []domain.StartTimer{
		{Name: "creation_timeout", After: time.Second, Pausable: true},
		{Name: "seat_deadline:1", After: time.Second, Pausable: true},
		{Name: "turn_timer", After: time.Second, Pausable: true},
		{Name: "idle_elapsed", After: time.Second, Pausable: true},
		{Name: "combat_cap", After: time.Second, Pausable: true},
	} {
		timers.Start(timer)
	}
	timers.SetTurnTimersEnabled(false)
	clk.Advance(time.Second)
	got := make(map[string]bool)
	for {
		env, ok := inbox.Receive(context.Background())
		if !ok {
			break
		}
		got[env.Event.(domain.TimerFired).Name] = true
		if len(got) == 2 {
			break
		}
	}
	if !got["idle_elapsed"] || !got["combat_cap"] || len(got) != 2 {
		t.Fatalf("fired timers = %#v", got)
	}
	timers.Start(domain.StartTimer{Name: "turn_timer", After: time.Second, Pausable: true})
	clk.Advance(time.Second)
	if len(inbox.queue) != 0 {
		t.Fatal("disabled turn timer fired")
	}
	timers.SetTurnTimersEnabled(true)
	timers.Start(domain.StartTimer{Name: "turn_timer", After: time.Second, Pausable: true})
	clk.Advance(time.Second)
	if _, ok := inbox.Receive(context.Background()); !ok {
		t.Fatal("re-enabled turn timer did not fire")
	}
}

func TestRoom_HostTimersOffCancelsPacingTimers(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox(4)
	timers := NewTimers(clk, inbox)
	engine := &roomEngine{effects: []domain.Effect{
		domain.StartTimer{Name: "creation_timeout", After: time.Second, Pausable: true},
		domain.StartTimer{Name: "idle_elapsed", After: time.Second, Pausable: true},
	}}
	room := NewRoom(engine, clk, nil, nil, nil, WithTimers(timers))
	room.process(context.Background(), domain.Envelope{Event: domain.Join{Seat: 1}})
	room.process(context.Background(), domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostTimersOff}})
	clk.Advance(time.Second)
	env, ok := inbox.Receive(context.Background())
	if !ok || env.Event.(domain.TimerFired).Name != "idle_elapsed" {
		t.Fatalf("timer after host off = %#v, %v", env.Event, ok)
	}
	if len(inbox.queue) != 0 {
		t.Fatal("creation timeout survived host off")
	}
}
