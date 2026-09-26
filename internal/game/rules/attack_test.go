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
