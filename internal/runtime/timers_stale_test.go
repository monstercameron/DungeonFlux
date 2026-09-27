package runtime

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// stoppedCallbackClock models AfterFunc callbacks that have already started
// when Stop is called. They can still run after a replacement has been armed.
type stoppedCallbackClock struct {
	clock.Clock
	callbacks []func()
}

func (c *stoppedCallbackClock) AfterFunc(d time.Duration, fn func()) clock.Timer {
	c.callbacks = append(c.callbacks, fn)
	return c.NewTimer(d)
}

func TestTimers_StaleCallbackCannotExpireReplacement(t *testing.T) {
	for _, tc := range []struct {
		name  string
		reset func(*Timers)
	}{
		{"replace", func(*Timers) {}},
		{"cancel", func(timers *Timers) { timers.Cancel(domain.CancelTimer{Name: "turn_timer"}) }},
		{"new run", func(timers *Timers) { timers.Reset() }},
		{"stop all", func(timers *Timers) { timers.StopAll() }},
		{"disable and enable", func(timers *Timers) { timers.SetTurnTimersEnabled(false); timers.SetTurnTimersEnabled(true) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clk := &stoppedCallbackClock{Clock: clock.NewFake(time.Unix(0, 0))}
			inbox := NewInbox(4)
			timers := NewTimers(clk, inbox)
			start := domain.StartTimer{Name: "turn_timer", After: 10 * time.Second, Pausable: true}
			timers.Start(start)
			tc.reset(timers)
			timers.Start(start)
			clk.callbacks[0]()
			if len(inbox.queue) != 0 {
				t.Fatal("old callback expired the replacement timer")
			}
			clk.callbacks[1]()
			if len(inbox.queue) != 1 {
				t.Fatal("replacement did not fire exactly once")
			}
			clk.callbacks[0]()
			clk.callbacks[1]()
			if len(inbox.queue) != 1 {
				t.Fatal("completed callback fired twice")
			}
		})
	}
}
