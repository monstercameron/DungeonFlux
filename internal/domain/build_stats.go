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
	// Equipment is the class's starting-equipment option A gear list.
	Equipment []EquipmentItem `json:"equipment,omitempty"`
}

// EquipmentItem is one piece of starting gear on a character's inventory.
type EquipmentItem struct {
	// Name is the item's display name.
	Name string `json:"name"`
	// Description is a short line of flavor or mechanical effect.
	Description string `json:"description,omitempty"`
	// Slot groups the item for display: "weapon", "armor", "shield", "gear", or "coin".
	Slot string `json:"slot,omitempty"`
	// Worn marks armor and shields the character has equipped.
	Worn bool `json:"worn,omitempty"`
}
