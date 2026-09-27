package creation

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func rollSeatOne(t *testing.T) *Machine {
	t.Helper()
	machine, err := New([]byte("rename-seed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []domain.Act{
		{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
		{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		{Seat: 1, Move: vocab.MoveClass, Arg: "wizard"},
	} {
		if _, err := machine.Step(step); err != nil {
			t.Fatalf("step %v: %v", step, err)
		}
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRollHero}); err != nil {
		t.Fatalf("roll_hero: %v", err)
	}
	return &machine
}

func TestMachine_Rename_AcceptedAfterRoll(t *testing.T) {
	machine := rollSeatOne(t)
	result, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "  Sable Wren  "})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("rename should be accepted, got %#v", result)
	}
	seat, _ := machine.Seat(1)
	if seat.Flavor.Name != "Sable Wren" {
		t.Fatalf("Flavor.Name = %q, want trimmed %q", seat.Flavor.Name, "Sable Wren")
	}
}

func TestMachine_Rename_RejectedBeforeRoll(t *testing.T) {
	machine, err := New([]byte("rename-seed-2"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "Too Soon"}); err == nil {
		t.Fatal("rename before roll_hero should be rejected")
	}
}

func TestMachine_Rename_RejectedAfterLock(t *testing.T) {
	machine := rollSeatOne(t)
	if _, err := machine.Step(domain.FlavorDone{Seat: 1, Flavor: domain.Flavor{Name: "Isolde", Hook: "H"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.PCLocked{Seat: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "Too Late"}); err == nil {
		t.Fatal("rename after lock should be rejected")
	}
	seat, _ := machine.Seat(1)
	if seat.Flavor.Name != "Isolde" {
		t.Fatalf("locked seat name changed to %q", seat.Flavor.Name)
	}
}

func TestMachine_Rename_RejectsUnknownSeat(t *testing.T) {
	machine := rollSeatOne(t)
	if _, err := machine.Step(domain.Act{Seat: 9, Move: vocab.MoveRename, Arg: "Ghost"}); err == nil {
		t.Fatal("rename for an unknown seat should be rejected")
	}
}

func TestMachine_Rename_RejectsEmptyAndOverlong(t *testing.T) {
	cases := []struct {
		name string
		arg  string
	}{
		{"empty", "   "},
		{"control_only", "\u0007\u0001"},
		{"too_long", strings.Repeat("x", MaxHeroNameLength+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			machine := rollSeatOne(t)
			if _, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: tc.arg}); err == nil {
				t.Fatalf("rename with %q should be rejected", tc.arg)
			}
		})
	}
}

func TestMachine_Rename_StripsControlAndMarkup(t *testing.T) {
	machine := rollSeatOne(t)
	result, err := machine.Step(domain.Act{Seat: 1, Move: vocab.MoveRename, Arg: "<b>Sa\u0007ble</b>  the {Wren}"})
	if err != nil {
		t.Fatalf("rename: %v", err)
	}
	if strings.ContainsAny(result.Seat.Flavor.Name, "<>{}") {
		t.Fatalf("rename left markup in %q", result.Seat.Flavor.Name)
	}
}

func TestSanitizeHeroName(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"trims", "  Wren  ", "Wren", false},
		{"collapses_internal_space", "Sa   ble", "Sa ble", false},
		{"empty", "   ", "", true},
		{"only_control", "\u0000\u001f\u007f", "", true},
		{"strips_markup", "<Wren>", "Wren", false},
		{"max_length_ok", strings.Repeat("a", MaxHeroNameLength), strings.Repeat("a", MaxHeroNameLength), false},
		{"over_length", strings.Repeat("a", MaxHeroNameLength+1), "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := SanitizeHeroName(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("SanitizeHeroName(%q) = %q, want error", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("SanitizeHeroName(%q) unexpected error: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("SanitizeHeroName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestMachine_FlavorFailed_UsesDeterministicFallbackName(t *testing.T) {
	machine := rollSeatOne(t)
	result, err := machine.Step(domain.FlavorFailed{Seat: 1})
	if err != nil {
		t.Fatalf("flavor failed: %v", err)
	}
	if !result.Accepted || strings.TrimSpace(result.Seat.Flavor.Name) == "" {
		t.Fatalf("flavor_failed should set a fallback name, got %#v", result)
	}
	if result.Seat.Flavor.Name == "Hero 1" {
		t.Fatal("flavor_failed must not fall back to a bare seat label")
	}

	other, err := New([]byte("rename-seed"))
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range []domain.Act{
		{Seat: 1, Move: vocab.MoveSpecies, Arg: "elf"},
		{Seat: 1, Move: vocab.MoveGender, Arg: "female"},
		{Seat: 1, Move: vocab.MoveClass, Arg: "wizard"},
	} {
		if _, err := other.Step(step); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := other.Step(domain.Act{Seat: 1, Move: vocab.MoveRollHero}); err != nil {
		t.Fatal(err)
	}
	repeat, err := other.Step(domain.FlavorFailed{Seat: 1})
	if err != nil {
		t.Fatal(err)
	}
	if repeat.Seat.Flavor.Name != result.Seat.Flavor.Name {
		t.Fatalf("same seed, species, and gender should yield the same fallback name: %q vs %q", repeat.Seat.Flavor.Name, result.Seat.Flavor.Name)
	}
}

func TestMachine_FlavorFailed_DoesNotOverwriteAGeneratedName(t *testing.T) {
	machine := rollSeatOne(t)
	if _, err := machine.Step(domain.FlavorDone{Seat: 1, Flavor: domain.Flavor{Name: "Isolde", Hook: "H"}}); err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step(domain.FlavorFailed{Seat: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Seat.Flavor.Name != "Isolde" {
		t.Fatalf("flavor_failed after flavor_done overwrote the name: %q", result.Seat.Flavor.Name)
	}
}

func TestMachine_CreationTimeout_NeverLeavesABareHeroLabel(t *testing.T) {
	machine, err := New([]byte("timeout-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.TimerFired{Name: "creation_timeout"}); err != nil {
		t.Fatal(err)
	}
	for _, seatID := range []domain.SeatID{1, 2} {
		seat, ok := machine.Seat(seatID)
		if !ok {
			t.Fatalf("seat %d missing", seatID)
		}
		if !seat.Locked || strings.TrimSpace(seat.Flavor.Name) == "" {
			t.Fatalf("seat %d not fully defaulted: %#v", seatID, seat)
		}
		if seat.Flavor.Name == "Hero "+string(rune('0'+seatID)) {
			t.Fatalf("seat %d fell back to a bare hero label", seatID)
		}
	}
}
