package rules

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/rulings"
)

// Class names the four demo templates.
type Class string

const (
	// Paladin is the Strength-based martial template.
	Paladin Class = "paladin"
	// Rogue is the Dexterity-based martial template.
	Rogue Class = "rogue"
	// Bard is the Dexterity-based light weapon template.
	Bard Class = "bard"
	// Cleric is the Strength-based mace template.
	Cleric Class = "cleric"
)

// AbilityScores holds the six ability scores.
type AbilityScores struct{ Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma int }

// Build is a deterministic level-one hero card and its combat statistics.
type Build struct {
	Class           Class
	Species, Gender string
	Abilities       AbilityScores
	HP, MaxHP, AC   int
	AttackAbility   string
	AttackBonus     int
	PersuasionBonus int
	Skills          []string
}

// BuildHero creates a constrained, seeded level-one hero.
func BuildHero(source *dice.Roller, class Class, species, gender string) (Build, error) {
	if source == nil {
		return Build{}, errors.New("dice source is nil")
	}
	if class != Paladin && class != Rogue && class != Bard && class != Cleric {
		return Build{}, errors.New("unknown class")
	}
	abilities := baseline(class)
	// Shuffle the non-fixed scores in a deterministic way. The fixed attack
	// ability and Charisma remain unchanged, preserving the demo odds.
	values := []int{8, 10, 12, 13, 14, 15}
	if class == Paladin || class == Rogue {
		values = []int{8, 10, 12, 13, 14}
	}
	for i := len(values) - 1; i > 0; i-- {
		draw, err := source.Roll(i + 1)
		if err != nil {
			return Build{}, err
		}
		values[i], values[draw.Face-1] = values[draw.Face-1], values[i]
	}
	applyRolled(class, &abilities, values)
	hp, ac := derived(class, abilities)
	return Build{Class: class, Species: species, Gender: gender, Abilities: abilities,
		HP: hp, MaxHP: hp, AC: ac, AttackAbility: attackAbility(class), AttackBonus: attackBonus(class),
		PersuasionBonus: rulings.AbilityModifier(abilities.Charisma) + rulings.ProficiencyBonus(1), Skills: skills(class)}, nil
}

// DrawClass applies R-D7: seat one draws from all classes; seat two is
// restricted to paladin/rogue when seat one drew bard or cleric.
func DrawClass(source *dice.Roller, seat int, first Class) (Class, error) {
	if source == nil {
		return "", errors.New("dice source is nil")
	}
	choices := []Class{Paladin, Rogue, Bard, Cleric}
	if seat == 2 && (first == Bard || first == Cleric) {
		choices = []Class{Paladin, Rogue}
	}
	if seat != 1 && seat != 2 {
		return "", errors.New("seat must be 1 or 2")
	}
	draw, err := source.Roll(len(choices))
	if err != nil {
		return "", err
	}
	return choices[draw.Face-1], nil
}

func baseline(class Class) AbilityScores {
	switch class {
	case Paladin:
		return AbilityScores{17, 10, 14, 8, 12, 14}
	case Rogue:
		return AbilityScores{8, 17, 14, 10, 12, 14}
	case Bard:
		return AbilityScores{8, 15, 14, 10, 14, 14}
	default:
		return AbilityScores{10, 12, 13, 9, 17, 14}
	}
}

func applyRolled(class Class, a *AbilityScores, values []int) {
	if class == Paladin {
		a.Dexterity, a.Intelligence, a.Wisdom = values[0], values[1], values[2]
		a.Constitution = values[3]
		return
	}
	if class == Rogue {
		a.Strength, a.Intelligence, a.Wisdom = values[0], values[1], values[2]
		a.Constitution = values[3]
		return
	}
	if class == Bard {
		a.Strength, a.Intelligence, a.Wisdom, a.Constitution = values[0], values[1], values[2], values[3]
		return
	}
	a.Dexterity, a.Intelligence, a.Wisdom, a.Constitution = values[0], values[1], values[2], values[3]
}

func derived(class Class, a AbilityScores) (int, int) {
	con := rulings.AbilityModifier(a.Constitution)
	dex := rulings.AbilityModifier(a.Dexterity)
	switch class {
	case Paladin:
		return 10 + con, 18
	case Rogue:
		return 8 + con + 2, 11 + dex
	case Bard:
		return 8 + con, 11 + dex
	default:
		return 8 + con, min(15+dex, 17)
	}
}

func attackAbility(class Class) string {
	if class == Paladin || class == Cleric {
		return "str"
	}
	return "dex"
}
func attackBonus(class Class) int {
	if class == Bard {
		return 4
	}
	if class == Cleric {
		return 2
	}
	return 5
}
func skills(class Class) []string {
	switch class {
	case Paladin:
		return []string{"athletics", "intimidation", "persuasion", "insight", "perception"}
	case Rogue:
		return []string{"sleight_of_hand", "stealth", "persuasion", "deception", "investigation", "acrobatics", "perception"}
	case Bard:
		return []string{"persuasion", "performance", "deception", "insight", "religion", "perception"}
	default:
		return []string{"insight", "religion", "persuasion", "medicine", "perception"}
	}
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
