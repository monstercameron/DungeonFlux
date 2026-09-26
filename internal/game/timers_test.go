package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestState_TurnTimersOffSuppressesCreationTimeout(t *testing.T) {
	state := New(domain.OneShot{}, []byte("timers-off"), WithTurnTimers(false))
	out := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	if state.TurnTimersEnabled() {
		t.Fatal("turn timers enabled after disabled configuration")
	}
	for _, effect := range out.Effects {
		if _, ok := effect.(domain.StartTimer); ok {
			t.Fatalf("disabled start emitted timer: %#v", effect)
		}
	}
	if state.View().Path != vocab.StateCreation {
		t.Fatalf("path = %q, want creation", state.View().Path)
	}
}

func TestState_TimersOffCancelsAndOnAllowsFutureTimers(t *testing.T) {
	state := New(domain.OneShot{}, []byte("toggle"))
	state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostStart}})
	off := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostTimersOff}})
	if state.TurnTimersEnabled() || countCancelTimers(off.Effects) != 4 {
		t.Fatalf("off state/effects = %v/%#v", state.TurnTimersEnabled(), off.Effects)
	}
	on := state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostCmd("TIMERS_ON")}})
	if !state.TurnTimersEnabled() || len(on.Effects) != 0 {
		t.Fatalf("on state/effects = %v/%#v", state.TurnTimersEnabled(), on.Effects)
	}
}

func countCancelTimers(effects []domain.Effect) int {
	count := 0
	for _, effect := range effects {
		if _, ok := effect.(domain.CancelTimer); ok {
			count++
		}
	}
	return count
}
