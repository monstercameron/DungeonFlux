package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_SpeciesGenderRollAndLocks(t *testing.T) {
	machine, err := New([]byte("creation-seed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, seat := range []domain.SeatID{2, 1} {
		if _, err := machine.Step(domain.Act{Seat: seat, Move: vocab.MoveSpecies, Arg: "elf"}); err != nil {
			t.Fatal(err)
		}
		if _, err := machine.Step(domain.Act{Seat: seat, Move: vocab.MoveGender, Arg: "female"}); err != nil {
			t.Fatal(err)
		}
		if _, err := machine.Step(domain.Act{Seat: seat, Move: vocab.MoveClass, Arg: "wizard"}); err != nil {
			t.Fatal(err)
		}
		result, err := machine.Step(domain.Act{Seat: seat, Move: vocab.MoveRollHero})
		if err != nil || len(result.Effects) != 1 {
			t.Fatalf("roll result=%#v err=%v", result, err)
		}
		if _, err := machine.Step(domain.FlavorDone{Seat: seat, Flavor: domain.Flavor{Name: "A", Hook: "H"}}); err != nil {
			t.Fatal(err)
		}
		if _, err := machine.Step(domain.PCLocked{Seat: seat}); err != nil {
			t.Fatal(err)
		}
	}
	if !machine.Complete() {
		t.Fatal("both locked seats should complete creation")
	}
	seat, _ := machine.Seat(1)
	if seat.Species != "elf" || seat.Gender != "female" || seat.Build.PersuasionBonus != 4 {
		t.Fatalf("seat state=%#v", seat)
	}
}

func TestMachine_TapOrderDoesNotChangeSeatBuilds(t *testing.T) {
	first, err := buildInOrder(t, []domain.SeatID{1, 2})
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildInOrder(t, []domain.SeatID{2, 1})
	if err != nil {
		t.Fatal(err)
	}
	for _, seat := range []domain.SeatID{1, 2} {
		a, _ := first.Seat(seat)
		b, _ := second.Seat(seat)
		if a.Class != b.Class || a.Build.Abilities != b.Build.Abilities || a.Build.HP != b.Build.HP || a.Build.AC != b.Build.AC {
			t.Fatalf("seat %d changed: %#v %#v", seat, a, b)
		}
	}
}

func TestMachine_RejectsInvalidEventsAndTimeoutDefaults(t *testing.T) {
	machine, err := New([]byte("timeout"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []domain.Event{
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "kobold"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "artificer"},
		domain.Act{Seat: 1, Move: vocab.MoveRollHero},
		domain.PCLocked{Seat: 1},
	}
	for _, event := range cases {
		if _, err := machine.Step(event); err == nil {
			t.Fatalf("event %#v accepted", event)
		}
	}
	result, err := machine.Step(domain.TimerFired{Name: creationTimeout})
	if err != nil || !result.Complete {
		t.Fatalf("timeout result=%#v err=%v", result, err)
	}
	for _, seat := range machine.Seats() {
		if !seat.Locked || seat.Build.Class == "" || seat.Species != "human" {
			t.Fatalf("default seat=%#v", seat)
		}
	}
}

func TestMachine_AllowsBothSeatsToChooseTheSameClass(t *testing.T) {
	machine, err := New([]byte("same-class"))
	if err != nil {
		t.Fatal(err)
	}
	for _, seat := range []domain.SeatID{1, 2} {
		for _, event := range []domain.Event{
			domain.Act{Seat: seat, Move: vocab.MoveSpecies, Arg: "human"},
			domain.Act{Seat: seat, Move: vocab.MoveGender, Arg: "nonbinary"},
			domain.Act{Seat: seat, Move: vocab.MoveClass, Arg: "wizard"},
			domain.Act{Seat: seat, Move: vocab.MoveRollHero},
		} {
			if _, err := machine.Step(event); err != nil {
				t.Fatalf("seat %d event %#v: %v", seat, event, err)
			}
		}
	}
	for _, seat := range machine.Seats() {
		if seat.Class != "wizard" {
			t.Fatalf("seat class = %q", seat.Class)
		}
	}
}

func TestMachine_FlavorRequiresRolledSeat(t *testing.T) {
	machine, err := New([]byte("flavor"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.FlavorDone{Seat: 1}); err == nil {
		t.Fatal("unbuilt seat accepted flavor")
	}
}

func buildInOrder(t *testing.T, order []domain.SeatID) (Machine, error) {
	t.Helper()
	machine, err := New([]byte("same-seed"))
	if err != nil {
		return Machine{}, err
	}
	for _, seat := range order {
		for _, event := range []domain.Event{
			domain.Act{Seat: seat, Move: vocab.MoveSpecies, Arg: "human"},
			domain.Act{Seat: seat, Move: vocab.MoveGender, Arg: "male"},
			domain.Act{Seat: seat, Move: vocab.MoveClass, Arg: "paladin"},
			domain.Act{Seat: seat, Move: vocab.MoveRollHero},
		} {
			if _, err := machine.Step(event); err != nil {
				return Machine{}, err
			}
		}
	}
	return machine, nil
}
