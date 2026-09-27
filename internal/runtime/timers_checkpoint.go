package runtime

import (
	"slices"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// timerCheckpoint contains values only: no live clock handles or entry pointers.
type timerCheckpoint struct {
	entries           []savedTimer
	turnTimersEnabled bool
}

type savedTimer struct {
	name             string
	after, remaining time.Duration
	pausable, paused bool
	scope            domain.Scope
}

func (t *Timers) checkpoint() timerCheckpoint {
	t.mu.Lock()
	defer t.mu.Unlock()
	point := timerCheckpoint{turnTimersEnabled: t.turnTimersEnabled}
	now := t.clk.Now()
	for _, entry := range t.byName {
		remaining := entry.remaining
		if !entry.paused {
			remaining = nonNegative(entry.deadline.Sub(now))
		}
		point.entries = append(point.entries, savedTimer{
			name: entry.name, after: entry.after, remaining: remaining,
			pausable: entry.pausable, paused: entry.paused, scope: entry.scope,
		})
	}
	slices.SortFunc(point.entries, func(a, b savedTimer) int { return strings.Compare(a.name, b.name) })
	return point
}

func (t *Timers) restore(point timerCheckpoint) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, entry := range t.byName {
		entry.timer.Stop()
	}
	t.byName = make(map[string]*timerEntry, len(point.entries))
	t.turnTimersEnabled = point.turnTimersEnabled
	now := t.clk.Now()
	for _, saved := range point.entries {
		entry := &timerEntry{
			name: saved.name, after: saved.after, remaining: saved.remaining,
			deadline: now.Add(saved.remaining), pausable: saved.pausable,
			paused: saved.paused, scope: saved.scope, generation: 1, runtimeGeneration: t.runtimeGeneration,
		}
		t.byName[entry.name] = entry
		t.armLocked(entry)
		if entry.paused {
			entry.timer.Stop()
		}
	}
}
