package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestTimers_PauseResume_preservesRemainingDuration(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox()
	timers := NewTimers(clk, inbox)
	timers.Start(domain.StartTimer{Name: "turn", After: 10 * time.Second, Pausable: true})
	clk.Advance(4 * time.Second)
	timers.PauseAll()
	clk.Advance(20 * time.Second)
	if len(inbox.queue) != 0 {
		t.Fatal("paused timer fired")
	}
	timers.ResumeAll()
	clk.Advance(6 * time.Second)
	if _, ok := inbox.Receive(context.Background()); !ok {
		t.Fatal("timer did not fire after remaining duration")
	}
}

func TestTimers_FreezeThaw_onlyPausesNamedTimer(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox(2)
	timers := NewTimers(clk, inbox)
	timers.Start(domain.StartTimer{Name: "frozen", After: 5 * time.Second, Pausable: true})
	timers.Start(domain.StartTimer{Name: "live", After: 5 * time.Second, Pausable: true})
	clk.Advance(time.Second)
	timers.Freeze(domain.FreezeTimer{Name: "frozen"})
	clk.Advance(4 * time.Second)
	env, ok := inbox.Receive(context.Background())
	if !ok || env.Event.(domain.TimerFired).Name != "live" {
		t.Fatalf("live timer event = %#v, %v", env.Event, ok)
	}
	timers.Thaw(domain.ThawTimer{Name: "frozen"})
	clk.Advance(4 * time.Second)
	if _, ok := inbox.Receive(context.Background()); !ok {
		t.Fatal("thawed timer did not fire")
	}
}

func TestTimers_CancelAndNonPausablePause(t *testing.T) {
	clk := clock.NewFake(time.Unix(0, 0))
	inbox := NewInbox()
	timers := NewTimers(clk, inbox)
	timers.Start(domain.StartTimer{Name: "cancelled", After: time.Second, Pausable: true})
	timers.Cancel(domain.CancelTimer{Name: "cancelled"})
	timers.Start(domain.StartTimer{Name: "fixed", After: time.Second, Pausable: false})
	timers.PauseAll()
	clk.Advance(time.Second)
	if _, ok := inbox.Receive(context.Background()); !ok {
		t.Fatal("non-pausable timer did not fire")
	}
	timers.StopAll()
}
