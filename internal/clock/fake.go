package clock

import (
	"sync"
	"time"
)

// Fake is a manually advanced clock. It is safe to use from concurrent test
// goroutines; callbacks run synchronously in the goroutine calling Advance.
type Fake struct {
	mu     sync.Mutex
	now    time.Time
	seq    uint64
	timers map[*fakeTimer]struct{}
}

// NewFake returns a fake clock starting at start.
func NewFake(start time.Time) *Fake {
	return &Fake{now: start, timers: make(map[*fakeTimer]struct{})}
}

// Now returns the fake clock's current time.
func (f *Fake) Now() time.Time {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.now
}

// Since returns the duration from t to the fake clock's current time.
func (f *Fake) Since(t time.Time) time.Duration { return f.Now().Sub(t) }

// NewTimer returns a timer that fires when Advance reaches its deadline.
func (f *Fake) NewTimer(d time.Duration) Timer { return f.newTimer(d, nil) }

// AfterFunc returns a timer that invokes fn when Advance reaches its deadline.
func (f *Fake) AfterFunc(d time.Duration, fn func()) Timer { return f.newTimer(d, fn) }

// Advance moves the clock forward and fires due timers in deadline order.
func (f *Fake) Advance(d time.Duration) {
	if d < 0 {
		return
	}
	f.mu.Lock()
	f.now = f.now.Add(d)
	for {
		due := f.nextDueLocked()
		if due == nil {
			f.mu.Unlock()
			return
		}
		due.active = false
		delete(f.timers, due)
		when, callback := due.deadline, due.callback
		f.mu.Unlock()
		if callback != nil {
			callback()
		} else {
			due.channel <- when
		}
		f.mu.Lock()
	}
}

func (f *Fake) newTimer(d time.Duration, callback func()) *fakeTimer {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	t := &fakeTimer{clock: f, channel: make(chan time.Time, 1), callback: callback, order: f.seq}
	t.deadline = f.now.Add(maxDuration(d))
	t.active = true
	f.timers[t] = struct{}{}
	return t
}

func (f *Fake) nextDueLocked() *fakeTimer {
	var due *fakeTimer
	for timer := range f.timers {
		if timer.deadline.After(f.now) {
			continue
		}
		if due == nil || timer.deadline.Before(due.deadline) ||
			(timer.deadline.Equal(due.deadline) && timer.order < due.order) {
			due = timer
		}
	}
	return due
}

func maxDuration(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

type fakeTimer struct {
	clock    *Fake
	channel  chan time.Time
	callback func()
	deadline time.Time
	order    uint64
	active   bool
}

func (t *fakeTimer) C() <-chan time.Time { return t.channel }

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if !t.active {
		return false
	}
	t.active = false
	delete(t.clock.timers, t)
	return true
}

func (t *fakeTimer) Reset(d time.Duration) bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	wasActive := t.active
	t.clock.seq++
	t.order = t.clock.seq
	t.deadline = t.clock.now.Add(maxDuration(d))
	t.active = true
	t.clock.timers[t] = struct{}{}
	return wasActive
}
