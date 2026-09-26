package runtime

import (
	"context"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type timerEntry struct {
	name       string
	after      time.Duration
	remaining  time.Duration
	deadline   time.Time
	pausable   bool
	paused     bool
	scope      domain.Scope
	timer      clock.Timer
	generation uint64
}

// Timers manages room timers using an injected clock. Pausable timers retain
// their remaining duration while paused and post TimerFired on expiry.
type Timers struct {
	mu     sync.Mutex
	clk    clock.Clock
	inbox  ports.Inbox
	byName map[string]*timerEntry
}

// NewTimers constructs a timer set.
func NewTimers(clk clock.Clock, inbox ports.Inbox) *Timers {
	if clk == nil {
		clk = clock.Real{}
	}
	return &Timers{clk: clk, inbox: inbox, byName: make(map[string]*timerEntry)}
}

// Start arms or replaces a named timer.
func (t *Timers) Start(effect domain.StartTimer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if old := t.byName[effect.Name]; old != nil {
		old.timer.Stop()
	}
	entry := &timerEntry{
		name: effect.Name, after: nonNegative(effect.After), remaining: nonNegative(effect.After),
		deadline: t.clk.Now().Add(nonNegative(effect.After)), pausable: effect.Pausable, scope: effect.Scope,
		generation: 1,
	}
	t.byName[entry.name] = entry
	t.armLocked(entry)
}

// Cancel stops and removes a named timer.
func (t *Timers) Cancel(effect domain.CancelTimer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.byName[effect.Name]
	if entry == nil {
		return
	}
	entry.timer.Stop()
	delete(t.byName, effect.Name)
}

// Freeze stops one pausable timer and records its remaining duration.
func (t *Timers) Freeze(effect domain.FreezeTimer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.freezeLocked(effect.Name)
}

// Thaw resumes one frozen timer with its recorded remaining duration.
func (t *Timers) Thaw(effect domain.ThawTimer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.byName[effect.Name]
	if entry == nil || !entry.paused {
		return
	}
	entry.paused = false
	entry.deadline = t.clk.Now().Add(entry.remaining)
	entry.generation++
	t.armLocked(entry)
}

// PauseAll freezes every pausable timer.
func (t *Timers) PauseAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for name, entry := range t.byName {
		if entry.pausable {
			t.freezeLocked(name)
		}
	}
}

// ResumeAll thaws every paused timer.
func (t *Timers) ResumeAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, entry := range t.byName {
		if entry.paused {
			entry.paused = false
			entry.deadline = t.clk.Now().Add(entry.remaining)
			entry.generation++
			t.armLocked(entry)
		}
	}
}

// StopAll stops and removes every timer.
func (t *Timers) StopAll() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, entry := range t.byName {
		entry.timer.Stop()
	}
	t.byName = make(map[string]*timerEntry)
}

func (t *Timers) freezeLocked(name string) {
	entry := t.byName[name]
	if entry == nil || !entry.pausable || entry.paused {
		return
	}
	entry.remaining = nonNegative(entry.deadline.Sub(t.clk.Now()))
	entry.timer.Stop()
	entry.paused = true
	entry.generation++
}

func (t *Timers) armLocked(entry *timerEntry) {
	generation := entry.generation
	entry.timer = t.clk.AfterFunc(entry.remaining, func() { t.fire(entry.name, generation) })
}

func (t *Timers) fire(name string, generation uint64) {
	t.mu.Lock()
	entry := t.byName[name]
	if entry == nil || entry.generation != generation || entry.paused {
		t.mu.Unlock()
		return
	}
	delete(t.byName, name)
	scope := entry.scope
	t.mu.Unlock()
	if t.inbox != nil {
		t.inbox.Post(context.Background(), domain.Envelope{Scope: scope, Event: domain.TimerFired{Name: name}})
	}
}

func nonNegative(value time.Duration) time.Duration {
	if value < 0 {
		return 0
	}
	return value
}
