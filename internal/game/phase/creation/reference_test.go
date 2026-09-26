package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func TestReferenceEffectFor_UsesLockedCharacterDetails(t *testing.T) {
	effect := referenceEffectFor(SeatState{Seat: 2, Species: "elf", Gender: "female", Class: rules.Rogue, Flavor: domain.Flavor{Name: "Mira", Look: "scarred", Hook: "river debt"}})
	if effect.Seat != 2 || effect.Name != "Mira" || effect.Class != "rogue" || effect.Flavor != "scarred river debt" {
		t.Fatalf("effect=%#v", effect)
	}
}

func TestMachine_LockEmitsReferenceWithoutWaiting(t *testing.T) {
	machine, err := New([]byte("reference"))
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []domain.Event{
		domain.Act{Seat: 1, Move: "species", Arg: "human"},
		domain.Act{Seat: 1, Move: "gender", Arg: "male"},
		domain.Act{Seat: 1, Move: "class", Arg: "rogue"},
		domain.Act{Seat: 1, Move: "roll_hero"},
		domain.FlavorDone{Seat: 1, Flavor: domain.Flavor{Name: "Corin", Look: "weathered", Hook: "a debt"}},
	} {
		if _, err := machine.Step(event); err != nil {
			t.Fatal(err)
		}
	}
	result, err := machine.Step(domain.PCLocked{Seat: 1})
	if err != nil || len(result.Effects) != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}
	effect, ok := result.Effects[0].(domain.GenerateCharacterReference)
	if !ok || effect.Name != "Corin" || effect.Seat != 1 {
		t.Fatalf("effect=%#v", result.Effects[0])
	}
	if result.Seat.Locked {
		return
	}
	t.Fatal("lock should complete without waiting for reference")
}
