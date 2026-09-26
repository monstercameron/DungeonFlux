package clock

import "time"

// Timer is the controllable portion of a time.Timer used by the application.
type Timer interface {
	C() <-chan time.Time
	Stop() bool
	Reset(d time.Duration) bool
}

// Clock supplies time and timers to code that must be testable.
type Clock interface {
	Now() time.Time
	Since(t time.Time) time.Duration
	NewTimer(d time.Duration) Timer
	AfterFunc(d time.Duration, f func()) Timer
}

// Real uses the operating system clock.
type Real struct{}

// Now returns the current wall-clock time.
func (Real) Now() time.Time { return time.Now() }

// Since returns the elapsed time since t.
func (Real) Since(t time.Time) time.Duration { return time.Since(t) }

// NewTimer returns a timer backed by time.NewTimer.
func (Real) NewTimer(d time.Duration) Timer { return realTimer{timer: time.NewTimer(d)} }

// AfterFunc returns a timer backed by time.AfterFunc.
func (Real) AfterFunc(d time.Duration, f func()) Timer {
	return realTimer{timer: time.AfterFunc(d, f)}
}

type realTimer struct{ timer *time.Timer }

func (t realTimer) C() <-chan time.Time { return t.timer.C }
func (t realTimer) Stop() bool          { return t.timer.Stop() }
func (t realTimer) Reset(d time.Duration) bool { return t.timer.Reset(d) }
