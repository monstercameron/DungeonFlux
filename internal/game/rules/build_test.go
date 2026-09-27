package rules

import (
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"testing"
)

func TestBuildHeroTemplatesAndDeterminism(t *testing.T) {
	wantHitDie := map[Class]int{Barbarian: 12, Bard: 8, Cleric: 8, Druid: 8, Fighter: 10, Monk: 8, Paladin: 10, Ranger: 10, Rogue: 8, Sorcerer: 6, Warlock: 8, Wizard: 6}
	wantAttack := map[Class]WeaponAttack{
		Barbarian: {"Greataxe", "1d12+3", "slashing", 5}, Bard: {"Dagger", "1d4+2", "piercing", 4},
		Cleric: {"Mace", "1d6", "bludgeoning", 2}, Druid: {"Scimitar", "1d6+3", "slashing", 5},
		Fighter: {"Longsword", "1d8+3", "slashing", 5}, Monk: {"Quarterstaff", "1d6+3", "bludgeoning", 5},
		Paladin: {"Longsword", "1d8+3", "slashing", 5}, Ranger: {"Longbow", "1d8+3", "piercing", 5},
		Rogue: {"Shortsword", "1d6+3", "piercing", 5}, Sorcerer: {"Dagger", "1d4+3", "piercing", 5},
		Warlock: {"Light Crossbow", "1d8+3", "piercing", 5}, Wizard: {"Dagger", "1d4+3", "piercing", 5},
	}
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
		if a.HitDie != wantHitDie[class] || a.HP < 11 || a.HP > 12 || a.Attack != wantAttack[class] || len(a.PrimaryAbilities) == 0 {
			t.Fatalf("incomplete %s template: %#v", class, a)
		}
		if len(a.SaveProficiencies) != 2 || a.SkillProficiencies["persuasion"] != "proficient" {
			t.Fatalf("missing proficiencies for %s: %#v", class, a)
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

func TestBuildHeroConstrainedRolls(t *testing.T) {
	for _, class := range Classes() {
		for seed := byte(0); seed < 32; seed++ {
			build, err := BuildHero(dice.New([]byte{seed}), class, "human", "nonbinary")
			if err != nil {
				t.Fatal(err)
			}
			if build.Abilities.Constitution < 12 {
				t.Fatalf("%s constitution=%d", class, build.Abilities.Constitution)
			}
			if build.Abilities.Dexterity == 8 {
				t.Fatalf("%s dexterity received the lowest roll", class)
			}
		}
	}
}
