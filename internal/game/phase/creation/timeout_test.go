package creation

import (
	"reflect"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_TimeoutPreservesChoicesAndLocksRolledHero(t *testing.T) {
	for _, tc := range []struct {
		name   string
		rolled bool
	}{
		{name: "partial choices"},
		{name: "rolled and named", rolled: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			machine, err := New([]byte("timeout-choices"))
			if err != nil {
				t.Fatal(err)
			}
			for _, event := range []domain.Event{
				domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
				domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
				domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "wizard"},
			} {
				if _, err := machine.Step(event); err != nil {
					t.Fatal(err)
				}
			}
			if tc.rolled {
				if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRollHero}); err != nil {
					t.Fatal(err)
				}
				if _, err := machine.Step(domain.FlavorDone{Seat: 1, Flavor: domain.Flavor{Name: "Sable", Look: "silver cloak", Hook: "river debt"}}); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := machine.Seat(1)
			result, err := machine.Step(domain.TimerFired{Name: creationTimeout})
			if err != nil || !result.Complete {
				t.Fatalf("timeout = %#v, %v", result, err)
			}
			first, _ := machine.Seat(1)
			if first.Species != "elf" || first.Gender != "female" || first.Class != "wizard" {
				t.Fatalf("player choices overwritten: %#v", first)
			}
			if tc.rolled && (!reflect.DeepEqual(first.Build, before.Build) || first.Flavor != before.Flavor) {
				t.Fatalf("rolled hero changed: before=%#v after=%#v", before, first)
			}
			for _, seat := range machine.Seats() {
				if !seat.Built || !seat.Locked || seat.Flavor.Name == "" || seat.Build.HP <= 0 {
					t.Fatalf("unfinished hero after timeout: %#v", seat)
				}
			}
			if len(result.Effects) != 2 {
				t.Fatalf("want one reference request per locked hero, got %#v", result.Effects)
			}
			for i, effect := range result.Effects {
				ref, ok := effect.(domain.GenerateCharacterReference)
				seat, _ := machine.Seat(domain.SeatID(i + 1))
				if !ok || ref.Seat != seat.Seat || ref.Name != seat.Flavor.Name || ref.Species != seat.Species {
					t.Fatalf("reference does not describe locked hero: %#v", effect)
				}
			}
			result, err = machine.Step(domain.TimerFired{Name: creationTimeout})
			if err != nil || !result.Complete || len(result.Effects) != 0 {
				t.Fatalf("repeated deadline resubmitted work: %#v, %v", result, err)
			}
		})
	}
}

func TestMachine_SeatTimeoutPreservesPartialChoices(t *testing.T) {
	machine, err := New([]byte("partial-seat"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.Act{Seat: 2, Move: vocab.MoveSpecies, Arg: "dwarf"}); err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step(domain.TimerFired{Name: "seat_deadline:2"})
	if err != nil || !result.Seat.Locked || result.Seat.Species != "dwarf" || result.Seat.Gender != "nonbinary" {
		t.Fatalf("deadline = %#v, %v", result, err)
	}
	first, _ := machine.Seat(1)
	if first.Built || first.Locked {
		t.Fatalf("other seat changed: %#v", first)
	}
}
