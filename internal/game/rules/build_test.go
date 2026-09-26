package rules

import (
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"testing"
)

func TestBuildHeroTemplatesAndDeterminism(t *testing.T) {
	for _, class := range Classes() {
		a, err := BuildHero(dice.New([]byte("same")), class, "human", "female")
		if err != nil {
			t.Fatal(err)
		}
		b, err := BuildHero(dice.New([]byte("same")), class, "human", "female")
		if err != nil {
			t.Fatal(err)
		}
		if a.Class != class || a.Abilities != b.Abilities || a.HP != b.HP || a.AC != b.AC || a.PersuasionBonus != 4 {
			t.Fatalf("%s mismatch: %#v %#v", class, a, b)
		}
		if a.Abilities.Charisma != 14 || a.PersuasionBonus != 4 || !a.PersuasionProficient || len(a.Skills) == 0 {
			t.Fatalf("invalid %s build: %#v", class, a)
		}
		if a.HitDie < 6 || a.HP < 11 || a.HP > 12 || a.Attack.Name == "" || len(a.PrimaryAbilities) == 0 {
			t.Fatalf("incomplete %s template: %#v", class, a)
		}
	}
}

func TestDrawClassRulingAndValidation(t *testing.T) {
	r := dice.New([]byte("class"))
	if _, err := DrawClass(r, 3, Paladin); err == nil {
		t.Fatal("invalid seat accepted")
	}
	if _, err := BuildHero(nil, Paladin, "", ""); err == nil {
		t.Fatal("nil source accepted")
	}
	if _, err := BuildHero(dice.New(nil), Class("not-a-class"), "", ""); err == nil {
		t.Fatal("invalid class accepted")
	}
	for i := 0; i < 20; i++ {
		class, err := DrawClass(r, 2, Bard)
		if err != nil {
			t.Fatal(err)
		}
		if class != Paladin && class != Rogue {
			t.Fatalf("class=%s", class)
		}
	}
}
