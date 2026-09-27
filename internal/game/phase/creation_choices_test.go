package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestView_CreationChoicesKeepInterleavedSeatsSeparateBeforeRoll(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("creation-choice-view"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Act{Seat: 2, Move: vocab.MoveSpecies, Arg: "orc"},
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
		domain.Act{Seat: 2, Move: vocab.MoveGender, Arg: "male"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "rogue"},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatalf("event %#v: %v", event, err)
		}
	}
	view := machine.View()
	if len(view.Seats) != 2 || view.Seats[0].Creation == nil || view.Seats[1].Creation == nil {
		t.Fatalf("creation choices missing: %#v", view.Seats)
	}
	first, second := view.Seats[0].Creation, view.Seats[1].Creation
	if first.Species != "elf" || first.Gender != "female" || first.Class != "rogue" || first.Rolled {
		t.Fatalf("seat one choice = %#v", first)
	}
	if second.Species != "orc" || second.Gender != "male" || second.Class != "" || second.Rolled {
		t.Fatalf("seat two choice = %#v", second)
	}
}

func TestView_CreationChoicesCarryDistinctRollAndReadyState(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("creation-choice-roll"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
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
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveReady}); err != nil {
		t.Fatal(err)
	}
	view := machine.View()
	if !view.Seats[0].Creation.Rolled || !view.Seats[1].Creation.Rolled {
		t.Fatalf("rolled choices = %#v", view.Seats)
	}
	if !view.Seats[0].Creation.Ready || view.Seats[1].Creation.Ready {
		t.Fatalf("ready choices = %#v", view.Seats)
	}
	if view.Seats[0].Build == nil || view.Seats[1].Build == nil || view.Seats[0].Build.Name == view.Seats[1].Build.Name {
		t.Fatalf("build cards = %#v", view.Seats)
	}
}
