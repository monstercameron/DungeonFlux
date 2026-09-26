package rules

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/rulings"
)

// Class names an SRD 5.2.1 class.
type Class string

const (
	// Barbarian is the Strength-based d12 template.
	Barbarian Class = "barbarian"
	// Bard is the Dexterity-based light weapon template.
	Bard Class = "bard"
	// Cleric is the Strength-based mace template.
	Cleric Class = "cleric"
	// Druid is the Dexterity-based scimitar template.
	Druid Class = "druid"
	// Fighter is the Strength-based martial template.
	Fighter Class = "fighter"
	// Monk is the Dexterity-based martial arts template.
	Monk Class = "monk"
	// Paladin is the Strength-based martial template.
	Paladin Class = "paladin"
	// Ranger is the Dexterity-based ranged template.
	Ranger Class = "ranger"
	// Rogue is the Dexterity-based martial template.
	Rogue Class = "rogue"
	// Sorcerer is the Charisma-focused template.
	Sorcerer Class = "sorcerer"
	// Warlock is the Charisma-focused template.
	Warlock Class = "warlock"
	// Wizard is the Intelligence-focused template.
	Wizard Class = "wizard"
)

// AbilityScores holds the six ability scores.
type AbilityScores struct{ Strength, Dexterity, Constitution, Intelligence, Wisdom, Charisma int }

// Build is a deterministic level-one hero card and its combat statistics.
type Build struct {
	Class                Class
	Species, Gender      string
	Abilities            AbilityScores
	HP, MaxHP, AC        int
	HitDie               int
	PrimaryAbilities     []string
	SaveProficiencies    []string
	SkillProficiencies   map[string]string
	AttackAbility        string
	AttackBonus          int
	Attack               WeaponAttack
	SneakAttack          bool
	PersuasionBonus      int
	PersuasionProficient bool
	PersuasionExpertise  bool
	PersuasionNote       string
	Skills               []string
}

// BuildHero creates a constrained, seeded level-one hero.
func BuildHero(source *dice.Roller, class Class, species, gender string) (Build, error) {
	if source == nil {
		return Build{}, errors.New("dice source is nil")
	}
	if !IsClass(class) {
		return Build{}, errors.New("unknown class")
	}
	template := templateFor(class)
	abilities := template.Baseline
	// Shuffle the non-fixed scores in a deterministic way. The fixed attack
	// ability and Charisma remain unchanged, preserving the demo odds.
	values := []int{8, 10, 12, 14}
	for i := len(values) - 1; i > 0; i-- {
		draw, err := source.Roll(i + 1)
		if err != nil {
			return Build{}, err
		}
		values[i], values[draw.Face-1] = values[draw.Face-1], values[i]
	}
	constrainValues(template.RollOrder, values)
	applyRolled(&abilities, template.RollOrder, values)
	hp, ac := derived(template, abilities)
	skills := make(map[string]string, len(template.Skills))
	for _, skill := range template.Skills {
		skills[skill] = "proficient"
	}
	for _, skill := range template.Expertise {
		skills[skill] = "expertise"
	}
	return Build{Class: class, Species: species, Gender: gender, Abilities: abilities,
		HP: hp, MaxHP: hp, AC: ac, HitDie: template.HitDie, PrimaryAbilities: append([]string(nil), template.PrimaryAbilities...),
		SaveProficiencies: append([]string(nil), template.SaveProficiencies...), SkillProficiencies: skills,
		AttackAbility: template.AttackAbility, AttackBonus: template.Attack.Bonus, Attack: template.Attack, SneakAttack: template.SneakAttack,
		PersuasionBonus: rulings.AbilityModifier(abilities.Charisma) + rulings.ProficiencyBonus(1), PersuasionProficient: true,
		PersuasionNote: template.PersuasionNote, Skills: append([]string(nil), template.Skills...)}, nil
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

func applyRolled(a *AbilityScores, order []string, values []int) {
	for index, ability := range order {
		value := values[index]
		switch ability {
		case "str":
			a.Strength = value
		case "dex":
			a.Dexterity = value
		case "con":
			a.Constitution = value
		case "int":
			a.Intelligence = value
		case "wis":
			a.Wisdom = value
		}
	}
}

func constrainValues(order []string, values []int) {
	con := indexOf(order, "con")
	if con >= 0 && values[con] < 12 {
		for index, value := range values {
			if value >= 12 {
				values[con], values[index] = values[index], values[con]
				break
			}
		}
	}
	dex := indexOf(order, "dex")
	if dex >= 0 && values[dex] == 8 {
		for index, value := range values {
			if value != 8 {
				values[dex], values[index] = values[index], values[dex]
				break
			}
		}
	}
}

func indexOf(values []string, wanted string) int {
	for index, value := range values {
		if value == wanted {
			return index
		}
	}
	return -1
}

func derived(template classTemplate, a AbilityScores) (int, int) {
	hp := template.HitDie + rulings.AbilityModifier(a.Constitution)
	if hp < 11 {
		hp = 11
	}
	if hp > 12 {
		hp = 12
	}
	return hp, template.AC(a)
}
