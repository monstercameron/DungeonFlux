package rulings

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

// Ruling identifies a named demo deviation or combat rule.
type Ruling string

const (
	// RulingD4 identifies the Down and stabilization rule.
	RulingD4 Ruling = "R-D4"
	// RulingD5 identifies the Rogue Sneak Attack rule.
	RulingD5 Ruling = "R-D5"
	// Ruling09 identifies critical damage dice doubling.
	Ruling09 Ruling = "R-09"
	// RulingClassTemplate identifies the all-class demo template adjustment.
	RulingClassTemplate Ruling = "R-CLASS-TEMPLATE"
)

// DemoRuling describes a rule that is useful to combat logs and review tools.
type DemoRuling struct {
	ID   Ruling
	Text string
}

// DemoRulings returns the combat and class-template rulings used by this
// package. A fresh slice is returned so callers can retain or annotate it.
func DemoRulings() []DemoRuling {
	return []DemoRuling{
		{ID: RulingD4, Text: "At zero HP a PC is Down and Prone; combat stabilizes it at 1 HP."},
		{ID: RulingD5, Text: "A Rogue adds 1d6 when the other PC is adjacent and the target is not Down."},
		{ID: Ruling09, Text: "A natural 20 doubles every damage die, including Sneak Attack."},
		{ID: RulingClassTemplate, Text: "The twelve-class demo fixes Persuasion proficiency and keeps HP in the 11-12 combat band."},
	}
}

// Term is one labelled contribution to a check modifier.
type Term struct {
	Label string
	Value int
}

// RollRecord stores the raw faces and kept face for a d20 test.
type RollRecord struct {
	RollID  string
	Counter uint64
	Die     int
	Faces   []int
	Kept    int
	Adv     int8
	Source  string
}

// CheckOutcome is the complete, replayable result of an ability check.
type CheckOutcome struct {
	CheckID   string
	Roll      RollRecord
	Modifier  int
	Breakdown []Term
	Total     int
	DC        int
	Success   bool
	Margin    int
	Natural   int
}

// AbilityModifier returns the SRD modifier for an ability score.
func AbilityModifier(score int) int {
	if score >= 10 {
		return (score - 10) / 2
	}
	return -((10 - score + 1) / 2)
}

// ProficiencyBonus returns the level-based proficiency bonus for levels 1–20.
func ProficiencyBonus(level int) int {
	if level < 1 {
		return 0
	}
	return 2 + (level-1)/4
}

// Check resolves a d20 ability check. Natural 1 and 20 are recorded for
// narration but do not override the normal total comparison.
func Check(source *dice.Roller, checkID string, modifier, dc int, adv int8, breakdown []Term) (CheckOutcome, error) {
	if source == nil {
		return CheckOutcome{}, errors.New("dice source is nil")
	}
	faces, kept, origin, err := source.RollD20(adv)
	if err != nil {
		return CheckOutcome{}, err
	}
	record := RollRecord{
		RollID: checkID + "-" + formatCounter(kept.Counter), Counter: kept.Counter,
		Die: 20, Faces: append([]int(nil), faces...), Kept: kept.Face, Adv: adv, Source: origin,
	}
	total := kept.Face + modifier
	natural := 0
	if kept.Face == 1 || kept.Face == 20 {
		natural = kept.Face
	}
	return CheckOutcome{CheckID: checkID, Roll: record, Modifier: modifier,
		Breakdown: append([]Term(nil), breakdown...), Total: total, DC: dc,
		Success: total >= dc, Margin: total - dc, Natural: natural}, nil
}

// Persuasion resolves the demo's Charisma-based Persuasion check.
func Persuasion(source *dice.Roller, checkID string, charisma int, proficient bool, dc int, adv int8) (CheckOutcome, error) {
	modifier := AbilityModifier(charisma)
	breakdown := []Term{{Label: "cha", Value: modifier}}
	if proficient {
		bonus := ProficiencyBonus(1)
		modifier += bonus
		breakdown = append(breakdown, Term{Label: "prof", Value: bonus})
	}
	return Check(source, checkID, modifier, dc, adv, breakdown)
}

func formatCounter(counter uint64) string {
	const digits = "0123456789"
	if counter == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for counter > 0 {
		i--
		buf[i] = digits[counter%10]
		counter /= 10
	}
	return string(buf[i:])
}
