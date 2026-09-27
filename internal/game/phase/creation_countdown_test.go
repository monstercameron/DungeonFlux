package phase

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_CreationCountdownTracksLogicalTimeAndPause(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("creation-countdown"))
	if err != nil {
		t.Fatal(err)
	}
	machine.SetTime(2 * time.Second)
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	machine.SetTime(9 * time.Second)
	timer := machine.View().Seats[0].TurnTimer
	if timer.Name != "creation_timeout" || timer.TotalMS != 30000 || timer.RemainingMS != 23000 || timer.Frozen {
		t.Fatalf("active timer = %#v", timer)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostPause}); err != nil {
		t.Fatal(err)
	}
	machine.SetTime(20 * time.Second)
	timer = machine.View().Seats[0].TurnTimer
	if timer.RemainingMS != 23000 || !timer.Frozen {
		t.Fatalf("paused timer advanced = %#v", timer)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostResume}); err != nil {
		t.Fatal(err)
	}
	machine.SetTime(24 * time.Second)
	timer = machine.View().Seats[0].TurnTimer
	if timer.RemainingMS != 19000 || timer.Frozen {
		t.Fatalf("resumed timer = %#v", timer)
	}
}

func TestMachine_CreationCountdownHiddenWhenTimersDisabled(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("creation-countdown-off"))
	if err != nil {
		t.Fatal(err)
	}
	machine.ConfigureTurnTimers(false)
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	if timer := machine.View().Seats[0].TurnTimer; timer != (domain.TimerView{}) {
		t.Fatalf("disabled timer leaked into view = %#v", timer)
	}
}
