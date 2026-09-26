package clock

import (
	"testing"
	"time"
)

func TestFakeAdvance_firesDeadlineOrder(t *testing.T) {
	start := time.Unix(10, 0)
	f := NewFake(start)
	order := make([]int, 0, 3)
	f.AfterFunc(3*time.Second, func() { order = append(order, 3) })
	f.AfterFunc(time.Second, func() { order = append(order, 1) })
	f.AfterFunc(2*time.Second, func() { order = append(order, 2) })
	f.Advance(3 * time.Second)
	if got, want := len(order), 3; got != want {
		t.Fatalf("callbacks = %d, want %d", got, want)
	}
	for i, want := range []int{1, 2, 3} {
		if order[i] != want {
			t.Fatalf("order[%d] = %d, want %d", i, order[i], want)
		}
	}
	if got := f.Now(); !got.Equal(start.Add(3 * time.Second)) {
		t.Fatalf("now = %v", got)
	}
}

func TestFakeTimer_channelStopAndReset(t *testing.T) {
	f := NewFake(time.Unix(0, 0))
	timer := f.NewTimer(2 * time.Second)
	if !timer.Stop() {
		t.Fatal("first Stop = false")
	}
	if timer.Stop() {
		t.Fatal("second Stop = true")
	}
	if timer.Reset(time.Second) {
		t.Fatal("Reset of stopped timer = true")
	}
	f.Advance(time.Second)
	select {
	case got := <-timer.C():
		if !got.Equal(time.Unix(1, 0)) {
			t.Fatalf("fired at %v", got)
		}
	default:
		t.Fatal("timer did not fire")
	}
	if timer.Stop() {
		t.Fatal("Stop after fire = true")
	}
}

func TestFakeAdvance_negativeDoesNothing(t *testing.T) {
	start := time.Unix(0, 0)
	f := NewFake(start)
	f.Advance(-time.Second)
	if !f.Now().Equal(start) {
		t.Fatalf("now moved to %v", f.Now())
	}
}

func TestReal_clockAndTimer(t *testing.T) {
	var c Real
	before := c.Now()
	if c.Since(before) < 0 {
		t.Fatal("Since returned negative duration")
	}
	timer := c.NewTimer(time.Hour)
	if !timer.Stop() {
		t.Fatal("real timer did not stop")
	}
	callback := c.AfterFunc(time.Hour, func() {})
	if !callback.Stop() {
		t.Fatal("real callback timer did not stop")
	}
}
