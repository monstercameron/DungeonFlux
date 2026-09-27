package rules

import (
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"testing"
)

func TestThrallSlamMathAndNaturalOne(t *testing.T) {
	r := dice.New([]byte("slam"))
	if err := r.ForceD20(10); err != nil {
		t.Fatal(err)
	}
	out, err := Slam(r, "a", "pc", 8, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Hit || out.Total < 2 || out.Total > 9 || out.HPAfter != 10-out.Total || out.Damage[0].Type != "bludgeoning" {
		t.Fatalf("slam=%#v", out)
	}
	if err := r.ForceD20(1); err != nil {
		t.Fatal(err)
	}
	miss, err := Slam(r, "b", "pc", 8, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if miss.Hit || miss.Natural != 1 || miss.HPAfter != 10 {
		t.Fatalf("natural one=%#v", miss)
	}
}

func TestCriticalAttackDoublesDamageAndConditions(t *testing.T) {
	r := dice.New([]byte("crit"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	out, err := Attack(r, "crit", "pc", "thrall", 5, 8, "1d8", 3, "slashing", 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Hit || !out.Crit || len(out.Damage[0].Faces) != 2 {
		t.Fatalf("critical=%#v", out)
	}
	creature := Thrall("t")
	ApplyDamage(&creature, 6)
	if creature.HP != 6 || !hasCondition(&creature, Bloodied) {
		t.Fatalf("bloodied=%#v", creature)
	}
	ApplyDamage(&creature, 6)
	if !hasCondition(&creature, Down) || !hasCondition(&creature, Defeated) {
		t.Fatalf("down=%#v", creature)
	}
	EndCombat(&creature, false)
	if creature.HP != 1 || hasCondition(&creature, Down) {
		t.Fatalf("stabilized=%#v", creature)
	}
}

func TestAttackValidationAndFlee(t *testing.T) {
	if _, err := Attack(nil, "", "", "", 0, 8, "bad", 0, "", 1, 0); err == nil {
		t.Fatal("nil source accepted")
	}
	if _, err := Attack(dice.New(nil), "", "", "", 0, 8, "1d0", 0, "", 1, 0); err == nil {
		t.Fatal("invalid damage accepted")
	}
	creature := Thrall("t")
	ApplyDamage(&creature, -1)
	EndCombat(&creature, true)
	if !hasCondition(&creature, Fled) || creature.HP != 12 {
		t.Fatalf("fled=%#v", creature)
	}
}

func TestAttackOddsAndSneakAttackRulings(t *testing.T) {
	classes := []struct {
		class  Class
		bonus  int
		want   int
		weapon string
	}{
		{Paladin, 5, 18, "1d8"}, {Rogue, 5, 18, "1d6"}, {Bard, 4, 17, "1d4"}, {Cleric, 2, 15, "1d6"},
	}
	for _, tc := range classes {
		t.Run(string(tc.class), func(t *testing.T) {
			hits := 0
			for face := 1; face <= 20; face++ {
				r := dice.New([]byte(tc.class))
				if err := r.ForceD20(face); err != nil {
					t.Fatal(err)
				}
				out, err := Attack(r, "odds", "pc", "thrall", tc.bonus, 8, tc.weapon, 0, "slashing", 12, 0)
				if err != nil {
					t.Fatal(err)
				}
				if out.Hit {
					hits++
				}
			}
			if hits != tc.want {
				t.Fatalf("hits=%d, want %d", hits, tc.want)
			}
		})
	}

	r := dice.New([]byte("sneak"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	out, err := Attack(r, "rogue", "pc", "thrall", 5, 8, "1d6", 3, "piercing", 12, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !CanSneakAttack(true, false) || CanSneakAttack(false, false) || CanSneakAttack(true, true) {
		t.Fatal("R-D5 eligibility mismatch")
	}
	if err := ApplySneakAttack(r, &out, true); err != nil {
		t.Fatal(err)
	}
	if len(out.Damage) != 2 || out.Damage[1].Dice != "2d6" || out.Damage[1].Source != "sneak_attack" {
		t.Fatalf("critical sneak damage=%#v", out.Damage)
	}
}
