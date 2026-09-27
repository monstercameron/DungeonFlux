package phase

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDebugTurn_SelectsLegalCombatTurn(t *testing.T) {
	for _, tc := range []struct {
		name  string
		turn  string
		phase string
		seat  int
	}{
		{name: "pc1", turn: "pc1", phase: "pc_turn", seat: 1},
		{name: "thrall", turn: "thrall", phase: "enemy_turn", seat: 1},
		{name: "pc2", turn: "pc2", phase: "pc_turn", seat: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-turn"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := machine.DebugGoto(vocab.StateCombat); err != nil {
				t.Fatal(err)
			}
			if err := machine.DebugTurn(tc.turn); err != nil {
				t.Fatal(err)
			}
			if got := string(machine.combat.Phase); got != tc.phase {
				t.Fatalf("phase = %q, want %q", got, tc.phase)
			}
			if machine.combat.TurnSeat != tc.seat {
				t.Fatalf("turn seat = %d, want %d", machine.combat.TurnSeat, tc.seat)
			}
			if tc.turn != "thrall" {
				active, ok := machine.combat.ActiveParticipant()
				if !ok || active.Seat != tc.seat {
					t.Fatalf("active participant = %#v/%v", active, ok)
				}
			}
		})
	}
}

func TestDebugTurn_RejectsInvalidWithoutMutation(t *testing.T) {
	machine, err := NewWithSeed(domain.OneShot{}, []byte("debug-turn-invalid"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.DebugGoto(vocab.StateCombat); err != nil {
		t.Fatal(err)
	}
	before := machine.combat
	if err := machine.DebugTurn("bogus"); err == nil {
		t.Fatal("invalid turn accepted")
	}
	if machine.combat.Phase != before.Phase || machine.combat.TurnSeat != before.TurnSeat || machine.combat.TurnNumber != before.TurnNumber {
		t.Fatalf("invalid turn mutated combat: before=%+v after=%+v", before, machine.combat)
	}
}
