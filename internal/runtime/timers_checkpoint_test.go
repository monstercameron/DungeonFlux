package runtime

import (
	"reflect"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestTimers_CheckpointRestoresRemainingAndPausedState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		paused bool
	}{
		{"running", false},
		{"paused", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := clock.NewFake(time.Unix(0, 0))
			inbox := NewInbox(8)
			timers := NewTimers(clk, inbox)
			scope := domain.Scope{Machine: "combat", Epoch: 7, Key: "hero"}
			timers.Start(domain.StartTimer{Name: "turn_timer", After: 10 * time.Second, Pausable: true, Scope: scope})
			clk.Advance(4 * time.Second)
			if tc.paused {
				timers.PauseAll()
			}
			point := timers.checkpoint()
			timers.StopAll()
			clk.Advance(time.Hour)
			for range 2 {
				timers.Start(domain.StartTimer{Name: "abandoned", After: time.Second})
				timers.restore(point)
				if !reflect.DeepEqual(timers.checkpoint(), point) {
					t.Fatal("restore changed saved timer values")
				}
				if tc.paused {
					clk.Advance(time.Minute)
					assertTimerQueueEmpty(t, inbox)
					timers.ResumeAll()
				}
				clk.Advance(5 * time.Second)
				assertTimerQueueEmpty(t, inbox)
				clk.Advance(time.Second)
				if len(inbox.queue) != 1 {
					t.Fatalf("expiry count = %d, want 1", len(inbox.queue))
				}
				env := <-inbox.queue
				if env.Scope != scope || env.Event.(domain.TimerFired).Name != "turn_timer" {
					t.Fatalf("wrong restored expiry: %#v", env)
				}
			}
		})
	}
}

func TestTimers_CheckpointRestoresPolicyWithoutChangingResetDefault(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox(8)
	timers := NewTimers(clk, inbox)
	timers.SetTurnTimersEnabled(false)
	timers.Start(domain.StartTimer{Name: "combat_cap", After: 10 * time.Second})
	point := timers.checkpoint()
	timers.SetTurnTimersEnabled(true)
	timers.restore(point)
	timers.Start(domain.StartTimer{Name: "turn_timer", After: time.Second, Pausable: true})
	timers.PauseAll()
	clk.Advance(10 * time.Second)
	if len(inbox.queue) != 1 || (<-inbox.queue).Event.(domain.TimerFired).Name != "combat_cap" {
		t.Fatal("restored policy must suppress pacing but retain the non-pausable cap")
	}
	timers.Reset()
	timers.Start(domain.StartTimer{Name: "turn_timer", After: time.Second, Pausable: true})
	clk.Advance(time.Second)
	if len(inbox.queue) != 1 {
		t.Fatal("restore changed the configured new-run default")
	}
}

func TestTimers_CheckpointRejectsAbandonedCallbacks(t *testing.T) {
	clk := &stoppedCallbackClock{Clock: clock.NewFake(time.Unix(0, 0))}
	inbox := NewInbox(8)
	timers := NewTimers(clk, inbox)
	timers.Start(domain.StartTimer{Name: "turn_timer", After: time.Second})
	point := timers.checkpoint()
	timers.restore(point)
	timers.restore(point)
	clk.callbacks[0]()
	clk.callbacks[1]()
	assertTimerQueueEmpty(t, inbox)
	clk.callbacks[2]()
	if len(inbox.queue) != 1 {
		t.Fatal("latest restored timer did not fire once")
	}
}

func TestTimers_CheckpointClampsOverdueAndRestoresEmpty(t *testing.T) {
	base := clock.NewFake(time.Unix(0, 0))
	clk := &stoppedCallbackClock{Clock: base}
	inbox := NewInbox(8)
	timers := NewTimers(clk, inbox)
	empty := timers.checkpoint()
	timers.Start(domain.StartTimer{Name: "overdue", After: time.Second})
	base.Advance(2 * time.Second)
	point := timers.checkpoint()
	if len(point.entries) != 1 || point.entries[0].remaining != 0 {
		t.Fatalf("overdue snapshot = %#v", point)
	}
	timers.restore(point)
	clk.callbacks[1]()
	if len(inbox.queue) != 1 {
		t.Fatal("overdue timer did not expire")
	}
	<-inbox.queue
	timers.Start(domain.StartTimer{Name: "abandoned", After: time.Second})
	timers.restore(empty)
	clk.callbacks[2]()
	assertTimerQueueEmpty(t, inbox)
	if len(timers.checkpoint().entries) != 0 {
		t.Fatal("empty checkpoint retained timers")
	}
}

func assertTimerQueueEmpty(t *testing.T, inbox *Inbox) {
	t.Helper()
	if len(inbox.queue) != 0 {
		t.Fatalf("unexpected expiry count: %d", len(inbox.queue))
	}
}
