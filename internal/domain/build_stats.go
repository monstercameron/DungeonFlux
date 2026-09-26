package domain

// BuildStats is the complete seeded level-one build shown to a client.
type BuildStats struct {
	// Abilities contains Strength, Dexterity, Constitution, Intelligence,
	// Wisdom, and Charisma, in that order.
	Abilities [6]int `json:"abilities"`
	// SaveProficiencies contains the proficient saving-throw ability IDs.
	SaveProficiencies []string `json:"save_proficiencies,omitempty"`
	// SkillProficiencies maps skill IDs to their proficiency level.
	SkillProficiencies map[string]string `json:"skill_proficiencies,omitempty"`
	// HP is the current hit-point total.
	HP int `json:"hp"`
	// MaxHP is the maximum hit-point total.
	MaxHP int `json:"max_hp"`
	// AC is the armor class.
	AC int `json:"ac"`
	// AttackName is the displayed primary weapon name.
	AttackName string `json:"attack_name"`
	// AttackDice is the displayed weapon damage expression.
	AttackDice string `json:"attack_dice"`
	// AttackDamageType is the displayed weapon damage type.
	AttackDamageType string `json:"attack_damage_type"`
	// AttackBonus is the primary weapon's attack modifier.
	AttackBonus int `json:"attack_bonus"`
}
