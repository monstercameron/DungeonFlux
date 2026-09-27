package fsm

import (
	"errors"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestTimers_pausableVirtualTime(t *testing.T) {
	timers := NewTimers()
	scope := Scope{Machine: vocab.MachineSession, Epoch: 2}
	started, err := timers.StartTimer(scope, "line", 5*time.Second, "speaking")
	if err != nil || started.Kind != vocab.EffectStartTimer {
		t.Fatalf("unexpected start: %#v, %v", started, err)
	}
	if _, err := timers.Advance(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	frozen, err := timers.FreezeTimer(scope, "line")
	if err != nil || frozen.Kind != vocab.EffectFreezeTimer || !frozen.Timer.Frozen {
		t.Fatalf("unexpected freeze: %#v, %v", frozen, err)
	}
	if _, err := timers.Advance(10 * time.Second); err != nil {
		t.Fatal(err)
	}
	if _, ok := timers.Get(scope, "line"); !ok {
		t.Fatal("frozen timer fired")
	}
	if _, err := timers.ThawTimer(scope, "line"); err != nil {
		t.Fatal(err)
	}
	fired, err := timers.Advance(3 * time.Second)
	if err != nil || len(fired) != 1 || fired[0].Name != "line" || fired[0].Stage != "speaking" || fired[0].Scope.Epoch != 2 {
		t.Fatalf("unexpected fired event: %#v, %v", fired, err)
	}
}

func TestTimers_cancelPreventsFiring(t *testing.T) {
	timers := NewTimers()
	scope := Scope{Machine: vocab.MachineCombat, Key: "1"}
	if _, err := timers.StartTimer(scope, "cap", time.Second); err != nil {
		t.Fatal(err)
	}
	cancelled, err := timers.CancelTimer(scope, "cap")
	if err != nil || cancelled.Kind != vocab.EffectCancelTimer {
		t.Fatalf("unexpected cancel: %#v, %v", cancelled, err)
	}
	fired, err := timers.Advance(time.Minute)
	if err != nil || len(fired) != 0 {
		t.Fatalf("cancelled timer fired: %#v, %v", fired, err)
	}
}

func TestTimers_rejectsInvalidOperations(t *testing.T) {
	timers := NewTimers()
	scope := Scope{Machine: vocab.MachineSession}
	tests := []struct {
		name string
		call func() error
		want string
	}{
		{name: "zero duration", call: func() error { _, err := timers.StartTimer(scope, "bad", 0); return err }, want: timerInvalid},
		{name: "negative duration", call: func() error { _, err := timers.StartTimer(scope, "bad", -time.Second); return err }, want: timerInvalid},
		{name: "missing cancel", call: func() error { _, err := timers.CancelTimer(scope, "missing"); return err }, want: timerMissing},
		{name: "missing freeze", call: func() error { _, err := timers.FreezeTimer(scope, "missing"); return err }, want: timerMissing},
		{name: "negative advance", call: func() error { _, err := timers.Advance(-time.Second); return err }, want: timerNegative},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var timerErr *TimerError
			if err := tc.call(); !errors.As(err, &timerErr) || timerErr.Reason != tc.want {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
	if _, err := timers.StartTimer(scope, "dup", time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := timers.StartTimer(scope, "dup", time.Second); err == nil {
		t.Fatal("duplicate timer accepted")
	}
}
