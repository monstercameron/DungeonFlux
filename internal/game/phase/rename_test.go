package phase

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// TestMachine_Rename_UpdatesPartyViewBeforeLock exercises the rename move end
// to end at the phase level: it is legal after roll_hero, it updates the
// projected Character/BuildCard name the DM and phone views read, and the
// engine rejects it once the seat locks.
func TestMachine_Rename_UpdatesPartyViewBeforeLock(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("rename-phase-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "human"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "nonbinary"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "paladin"},
		domain.Act{Seat: 1, Move: vocab.MoveRollHero},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatalf("event %T: %v", event, err)
		}
	}
	before := machine.View().Seats[0]
	if before.Character == nil || before.Build == nil {
		t.Fatalf("seat 1 view missing after roll: %#v", before)
	}

	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "  Sable Wren  "}); err != nil {
		t.Fatalf("rename: %v", err)
	}
	renamed := machine.View().Seats[0]
	if renamed.Character.Name != "Sable Wren" || renamed.Build.Name != "Sable Wren" {
		t.Fatalf("rename did not update the projected view: %#v", renamed)
	}

	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveReady}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "Too Late"}); err == nil {
		t.Fatal("rename after the seat locks should be rejected")
	}
	locked := machine.View().Seats[0]
	if locked.Character.Name != "Sable Wren" {
		t.Fatalf("locked seat name changed to %q", locked.Character.Name)
	}
}

// TestMachine_FlavorFailed_NeverShowsABareHeroLabel exercises the
// character_flavor failure path at the phase level: FlavorFailed is passive
// in every other phase, but Creation must still apply the curated fallback
// name so the party view and combat tokens never show "Hero 1".
func TestMachine_FlavorFailed_NeverShowsABareHeroLabel(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("flavor-failed-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Act{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
		domain.Act{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		domain.Act{Seat: 1, Move: vocab.MoveClass, Arg: "wizard"},
		domain.Act{Seat: 1, Move: vocab.MoveRollHero},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatalf("event %T: %v", event, err)
		}
	}
	if _, err := machine.Step(domain.FlavorFailed{Seat: 1}); err != nil {
		t.Fatalf("flavor_failed: %v", err)
	}
	seat := machine.View().Seats[0]
	if seat.Character == nil || strings.TrimSpace(seat.Character.Name) == "" {
		t.Fatalf("seat 1 has no name after flavor_failed: %#v", seat)
	}
	if seat.Character.Name == "Hero 1" {
		t.Fatal("flavor_failed must not leave the bare seat label \"Hero 1\"")
	}
}
