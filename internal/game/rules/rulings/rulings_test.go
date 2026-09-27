package rulings

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

func TestPersuasionFourVsTenHasSeventyFivePercentSuccess(t *testing.T) {
	r := dice.New([]byte("odds"))
	successes := 0
	for i := 1; i <= 20; i++ {
		if err := r.ForceD20(i); err != nil {
			t.Fatal(err)
		}
		outcome, err := Persuasion(r, "check", 14, true, 10, 0)
		if err != nil {
			t.Fatal(err)
		}
		if outcome.Success {
			successes++
		}
		if outcome.Modifier != 4 || outcome.Margin != outcome.Total-10 || len(outcome.Breakdown) != 2 {
			t.Fatalf("bad outcome: %#v", outcome)
		}
	}
	if successes != 15 {
		t.Fatalf("successes=%d, want 15", successes)
	}
}

func TestCheckRecordsForcedNaturalAndAdvantage(t *testing.T) {
	r := dice.New([]byte("record"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	outcome, err := Check(r, "c-1", -10, 15, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Success || outcome.Natural != 20 || outcome.Roll.Source != "FORCED" || outcome.Roll.Kept != 20 || len(outcome.Roll.Faces) != 2 || outcome.Roll.Counter != 0 || r.Counter() != 2 {
		t.Fatalf("record mismatch: %#v", outcome)
	}
}

func TestRulingsValidationAndModifiers(t *testing.T) {
	if AbilityModifier(9) != -1 || AbilityModifier(10) != 0 || AbilityModifier(17) != 3 {
		t.Fatal("ability modifiers incorrect")
	}
	if ProficiencyBonus(1) != 2 || ProficiencyBonus(5) != 3 || ProficiencyBonus(9) != 4 || ProficiencyBonus(0) != 0 {
		t.Fatal("proficiency bonuses incorrect")
	}
	if _, err := Check(nil, "x", 0, 10, 0, nil); err == nil {
		t.Fatal("nil source should fail")
	}
	if _, err := Check(dice.New(nil), "x", 0, 10, 2, nil); err == nil {
		t.Fatal("invalid advantage should fail")
	}
}

func TestDemoRulingsExposeCombatSurface(t *testing.T) {
	rows := DemoRulings()
	if len(rows) != 5 || rows[0].ID != RulingD4 || rows[1].ID != RulingD5 || rows[2].ID != Ruling09 || rows[3].ID != RulingClassTemplate || rows[4].ID != RulingD8 {
		t.Fatalf("rulings=%#v", rows)
	}
	for _, row := range rows {
		if row.Text == "" {
			t.Fatalf("empty ruling=%#v", row)
		}
	}
}
