package phase

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_TurnTimersOffSuppressesCreationTimeout(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("timers-off"))
	if err != nil {
		t.Fatal(err)
	}
	machine.ConfigureTurnTimers(false)
	result, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart})
	if err != nil {
		t.Fatal(err)
	}
	for _, effect := range result.Effects {
		if _, ok := effect.(domain.StartTimer); ok {
			t.Fatalf("disabled start emitted timer: %#v", effect)
		}
	}
	if _, err := machine.Step(domain.TimerFired{Name: "creation_timeout"}); err == nil {
		t.Fatal("disabled creation timeout was accepted")
	}
}

func TestMachine_RollStartsSeatDeadlineAndReadyCancelsIt(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("seat-deadline"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "human"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "paladin"},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatal(err)
		}
	}
	rolled, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRollHero})
	if err != nil {
		t.Fatal(err)
	}
	if !hasTimer(rolled.Effects, "seat_deadline:1", 22*time.Second) {
		t.Fatalf("roll effects = %#v", rolled.Effects)
	}
	ready, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveReady})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCancel(ready.Effects, "seat_deadline:1") {
		t.Fatalf("ready effects = %#v", ready.Effects)
	}
}

func hasTimer(effects []domain.Effect, name string, after time.Duration) bool {
	for _, effect := range effects {
		timer, ok := effect.(domain.StartTimer)
		if ok && timer.Name == name && timer.After == after && timer.Pausable {
			return true
		}
	}
	return false
}

func hasCancel(effects []domain.Effect, name string) bool {
	for _, effect := range effects {
		timer, ok := effect.(domain.CancelTimer)
		if ok && timer.Name == name {
			return true
		}
	}
	return false
}
