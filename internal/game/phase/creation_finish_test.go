package phase

import (
	"reflect"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_CreationFallbackProjectsBothHeroes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		timers bool
		event  domain.Event
	}{
		{name: "timeout", timers: true, event: domain.TimerFired{Name: "creation_timeout"}},
		{name: "skip", timers: true, event: domain.HostCmd{Cmd: vocab.HostSkip}},
		{name: "skip with timers off", event: domain.HostCmd{Cmd: vocab.HostSkip}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			machine, err := NewWithSeed(domain.OneShot{}, []byte("project-fallback"))
			if err != nil {
				t.Fatal(err)
			}
			machine.ConfigureTurnTimers(tc.timers)
			if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
				t.Fatal(err)
			}
			result, err := machine.Step(tc.event)
			if err != nil || machine.State() != vocab.StateOpening {
				t.Fatalf("finish creation = %#v, %v; state=%s", result, err, machine.State())
			}
			opening := machine.View()
			assertFinishedHeroes(t, opening)
			if err := machine.Goto(vocab.StateCombat); err != nil {
				t.Fatal(err)
			}
			battle := machine.View()
			assertFinishedHeroes(t, battle)
			for i, seat := range opening.Seats {
				if !reflect.DeepEqual(seat.Character, battle.Seats[i].Character) || !reflect.DeepEqual(seat.Build, battle.Seats[i].Build) {
					t.Fatalf("hero %d changed on combat entry: opening=%#v combat=%#v", i+1, seat, battle.Seats[i])
				}
			}
		})
	}
}

func assertFinishedHeroes(t *testing.T, view domain.View) {
	t.Helper()
	if len(view.Seats) != 2 {
		t.Fatalf("seats=%d", len(view.Seats))
	}
	for _, seat := range view.Seats {
		if seat.Character == nil || seat.Build == nil || seat.Build.Stats == nil {
			t.Fatalf("missing character sheet for seat %d: %#v", seat.Seat, seat)
		}
		if seat.Character.Name == "" || seat.Character.Class == "" || seat.Character.HP <= 0 || seat.Build.Stats.MaxHP != seat.Character.MaxHP || seat.Build.Stats.AttackName == "" {
			t.Fatalf("incomplete hero %d: character=%#v stats=%#v", seat.Seat, seat.Character, seat.Build.Stats)
		}
		for _, score := range seat.Build.Stats.Abilities {
			if score <= 0 {
				t.Fatalf("missing ability for hero %d", seat.Seat)
			}
		}
	}
}
