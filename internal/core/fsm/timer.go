package fsm

import (
	"fmt"
	"sort"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Timer describes one virtual timer owned by a scope.
type Timer struct {
	Scope     Scope
	Name      string
	Stage     string
	Remaining time.Duration
	Frozen    bool
}

// TimerFired is the pure event emitted when virtual time reaches a timer.
type TimerFired struct {
	Scope Scope
	Name  string
	Stage string
}

// TimerEffect records a timer command for the runtime executor.
type TimerEffect struct {
	Kind  vocab.EffectKind
	Timer Timer
}

// TimerError reports an invalid timer operation.
type TimerError struct {
	Name   string
	Reason string
}

func (e *TimerError) Error() string {
	return fmt.Sprintf("timer %q: %s", e.Name, e.Reason)
}

const (
	timerInvalid  = "invalid_duration"
	timerMissing  = "missing"
	timerExists   = "already_exists"
	timerNegative = "negative_advance"
)

type timerKey struct {
	scope ScopeID
	name  string
}

// Timers is a virtual-time timer book. It never sleeps or reads a wall clock.
type Timers struct {
	active map[timerKey]Timer
}

// NewTimers creates an empty virtual timer book.
func NewTimers() Timers {
	return Timers{active: make(map[timerKey]Timer)}
}

// StartTimer adds a timer and returns its start effect. An optional stage is
// copied to TimerFired; omitting it leaves the stage empty.
func (t *Timers) StartTimer(scope Scope, name string, duration time.Duration, stage ...string) (TimerEffect, error) {
	if name == "" || duration <= 0 {
		return TimerEffect{}, &TimerError{Name: name, Reason: timerInvalid}
	}
	key := timerKey{scope: ScopeID{Machine: scope.Machine, Key: scope.Key}, name: name}
	if _, exists := t.active[key]; exists {
		return TimerEffect{}, &TimerError{Name: name, Reason: timerExists}
	}
	value := ""
	if len(stage) > 0 {
		value = stage[0]
	}
	timer := Timer{Scope: scope, Name: name, Stage: value, Remaining: duration}
	t.active[key] = timer
	return TimerEffect{Kind: vocab.EffectStartTimer, Timer: timer}, nil
}

// CancelTimer removes a timer and returns its cancel effect.
func (t *Timers) CancelTimer(scope Scope, name string) (TimerEffect, error) {
	timer, key, err := t.lookup(scope, name)
	if err != nil {
		return TimerEffect{}, err
	}
	delete(t.active, key)
	return TimerEffect{Kind: vocab.EffectCancelTimer, Timer: timer}, nil
}

// FreezeTimer pauses a timer without changing its remaining duration.
func (t *Timers) FreezeTimer(scope Scope, name string) (TimerEffect, error) {
	timer, key, err := t.lookup(scope, name)
	if err != nil {
		return TimerEffect{}, err
	}
	timer.Frozen = true
	t.active[key] = timer
	return TimerEffect{Kind: vocab.EffectFreezeTimer, Timer: timer}, nil
}

// ThawTimer resumes a frozen timer.
func (t *Timers) ThawTimer(scope Scope, name string) (TimerEffect, error) {
	timer, key, err := t.lookup(scope, name)
	if err != nil {
		return TimerEffect{}, err
	}
	timer.Frozen = false
	t.active[key] = timer
	return TimerEffect{Kind: vocab.EffectThawTimer, Timer: timer}, nil
}

// Advance moves virtual time and returns every timer that reaches zero in
// deterministic insertion-independent key order. Fired timers are removed.
func (t *Timers) Advance(delta time.Duration) ([]TimerFired, error) {
	if delta < 0 {
		return nil, &TimerError{Reason: timerNegative}
	}
	var fired []TimerFired
	keys := make([]timerKey, 0, len(t.active))
	for key := range t.active {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].scope.Machine != keys[j].scope.Machine {
			return keys[i].scope.Machine < keys[j].scope.Machine
		}
		if keys[i].scope.Key != keys[j].scope.Key {
			return keys[i].scope.Key < keys[j].scope.Key
		}
		return keys[i].name < keys[j].name
	})
	for _, key := range keys {
		timer, exists := t.active[key]
		if !exists {
			continue
		}
		if timer.Frozen {
			continue
		}
		timer.Remaining -= delta
		if timer.Remaining > 0 {
			t.active[key] = timer
			continue
		}
		delete(t.active, key)
		fired = append(fired, TimerFired{Scope: timer.Scope, Name: timer.Name, Stage: timer.Stage})
	}
	return fired, nil
}

// Get returns a timer's current virtual state.
func (t *Timers) Get(scope Scope, name string) (Timer, bool) {
	key := timerKey{scope: ScopeID{Machine: scope.Machine, Key: scope.Key}, name: name}
	timer, exists := t.active[key]
	return timer, exists
}

func (t *Timers) lookup(scope Scope, name string) (Timer, timerKey, error) {
	key := timerKey{scope: ScopeID{Machine: scope.Machine, Key: scope.Key}, name: name}
	timer, exists := t.active[key]
	if !exists {
		return Timer{}, key, &TimerError{Name: name, Reason: timerMissing}
	}
	return timer, key, nil
}
