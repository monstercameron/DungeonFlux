package rules

// WeaponAttack describes the one level-one attack shown for a demo build.
type WeaponAttack struct {
	Name       string
	Dice       string
	DamageType string
	Bonus      int
}

type classTemplate struct {
	HitDie            int
	HPBonus           int
	Baseline          AbilityScores
	RollOrder         []string
	PrimaryAbilities  []string
	AttackAbility     string
	Attack            WeaponAttack
	SaveProficiencies []string
	Skills            []string
	Expertise         []string
	PersuasionNote    string
	SneakAttack       bool
	AC                func(AbilityScores) int
}

// Classes returns the twelve SRD 5.2.1 class identifiers in display order.
func Classes() []Class {
	return []Class{Barbarian, Bard, Cleric, Druid, Fighter, Monk, Paladin, Ranger, Rogue, Sorcerer, Warlock, Wizard}
}

// IsClass reports whether class is one of the twelve SRD 5.2.1 classes.
func IsClass(class Class) bool {
	switch class {
	case Barbarian, Bard, Cleric, Druid, Fighter, Monk, Paladin, Ranger, Rogue, Sorcerer, Warlock, Wizard:
		return true
	default:
		return false
	}
}

func templateFor(class Class) classTemplate {
	switch class {
	case Barbarian:
		return classTemplate{HitDie: 12, Baseline: AbilityScores{17, 10, 14, 8, 12, 14}, RollOrder: rollOrder("dex", "int", "wis", "con"), PrimaryAbilities: []string{"str", "con"}, AttackAbility: "str", Attack: WeaponAttack{"Greataxe", "1d12+3", "slashing", 5}, SaveProficiencies: []string{"str", "con"}, Skills: []string{"animal_handling", "athletics", "intimidation", "nature", "survival", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: barbarianAC}
	case Bard:
		return classTemplate{HitDie: 8, Baseline: AbilityScores{8, 15, 14, 10, 14, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"cha", "dex"}, AttackAbility: "dex", Attack: WeaponAttack{"Dagger", "1d4+2", "piercing", 4}, SaveProficiencies: []string{"dex", "cha"}, Skills: []string{"persuasion", "performance", "deception", "insight", "religion", "perception"}, PersuasionNote: "class skill", AC: lightArmorAC}
	case Cleric:
		return classTemplate{HitDie: 8, Baseline: AbilityScores{10, 12, 13, 9, 17, 14}, RollOrder: rollOrder("dex", "int", "wis", "con"), PrimaryAbilities: []string{"wis", "con"}, AttackAbility: "str", Attack: WeaponAttack{"Mace", "1d6", "bludgeoning", 2}, SaveProficiencies: []string{"wis", "cha"}, Skills: []string{"insight", "religion", "persuasion", "medicine", "perception"}, PersuasionNote: "class skill", AC: clericAC}
	case Druid:
		return classTemplate{HitDie: 8, Baseline: AbilityScores{10, 17, 13, 8, 15, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"wis", "con"}, AttackAbility: "dex", Attack: WeaponAttack{"Scimitar", "1d6+3", "slashing", 5}, SaveProficiencies: []string{"int", "wis"}, Skills: []string{"animal_handling", "insight", "medicine", "nature", "perception", "religion", "survival", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: lightArmorAC}
	case Fighter:
		return classTemplate{HitDie: 10, Baseline: AbilityScores{17, 10, 14, 8, 12, 14}, RollOrder: rollOrder("dex", "int", "wis", "con"), PrimaryAbilities: []string{"str", "con"}, AttackAbility: "str", Attack: WeaponAttack{"Longsword", "1d8+3", "slashing", 5}, SaveProficiencies: []string{"str", "con"}, Skills: []string{"athletics", "intimidation", "perception", "survival", "insight", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: heavyArmorAC}
	case Monk:
		return classTemplate{HitDie: 8, Baseline: AbilityScores{10, 17, 14, 8, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"dex", "wis"}, AttackAbility: "dex", Attack: WeaponAttack{"Quarterstaff", "1d6+3", "bludgeoning", 5}, SaveProficiencies: []string{"str", "dex"}, Skills: []string{"acrobatics", "athletics", "history", "insight", "religion", "stealth", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: monkAC}
	case Paladin:
		return classTemplate{HitDie: 10, Baseline: AbilityScores{17, 10, 14, 8, 12, 14}, RollOrder: rollOrder("dex", "int", "wis", "con"), PrimaryAbilities: []string{"str", "cha"}, AttackAbility: "str", Attack: WeaponAttack{"Longsword", "1d8+3", "slashing", 5}, SaveProficiencies: []string{"wis", "cha"}, Skills: []string{"athletics", "intimidation", "perception", "survival", "insight", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: heavyArmorAC}
	case Ranger:
		return classTemplate{HitDie: 10, Baseline: AbilityScores{10, 17, 14, 8, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"dex", "wis"}, AttackAbility: "dex", Attack: WeaponAttack{"Longbow", "1d8+3", "piercing", 5}, SaveProficiencies: []string{"str", "dex"}, Skills: []string{"animal_handling", "athletics", "insight", "investigation", "nature", "perception", "stealth", "survival", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: lightArmorAC}
	case Rogue:
		return classTemplate{HitDie: 8, HPBonus: 2, Baseline: AbilityScores{8, 17, 14, 10, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"dex", "int"}, AttackAbility: "dex", Attack: WeaponAttack{"Shortsword", "1d6+3", "piercing", 5}, SaveProficiencies: []string{"dex", "int"}, Skills: []string{"sleight_of_hand", "stealth", "persuasion", "deception", "investigation", "acrobatics", "perception"}, Expertise: []string{"sleight_of_hand", "stealth"}, PersuasionNote: "class skill; expertise excludes Persuasion", SneakAttack: true, AC: lightArmorAC}
	case Sorcerer:
		return classTemplate{HitDie: 6, Baseline: AbilityScores{10, 17, 13, 8, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"cha", "con"}, AttackAbility: "dex", Attack: WeaponAttack{"Dagger", "1d4+3", "piercing", 5}, SaveProficiencies: []string{"con", "cha"}, Skills: []string{"arcana", "deception", "insight", "intimidation", "persuasion", "religion"}, PersuasionNote: "class skill", AC: noArmorAC}
	case Warlock:
		return classTemplate{HitDie: 8, Baseline: AbilityScores{10, 17, 13, 8, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"cha", "con"}, AttackAbility: "dex", Attack: WeaponAttack{"Light Crossbow", "1d8+3", "piercing", 5}, SaveProficiencies: []string{"wis", "cha"}, Skills: []string{"arcana", "deception", "history", "intimidation", "investigation", "nature", "religion", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: lightArmorAC}
	case Wizard:
		return classTemplate{HitDie: 6, Baseline: AbilityScores{10, 17, 13, 15, 12, 14}, RollOrder: rollOrder("str", "int", "wis", "con"), PrimaryAbilities: []string{"int", "dex"}, AttackAbility: "dex", Attack: WeaponAttack{"Dagger", "1d4+3", "piercing", 5}, SaveProficiencies: []string{"int", "wis"}, Skills: []string{"arcana", "history", "insight", "investigation", "medicine", "nature", "religion", "persuasion"}, PersuasionNote: "demo-fixed proficiency; not a class skill", AC: noArmorAC}
	default:
		return classTemplate{}
	}
}

func rollOrder(values ...string) []string { return values }

func noArmorAC(a AbilityScores) int { return 10 + abilityMod(a.Dexterity) }

func lightArmorAC(a AbilityScores) int { return 11 + abilityMod(a.Dexterity) }

func heavyArmorAC(AbilityScores) int { return 18 }

func clericAC(a AbilityScores) int {
	value := 13 + abilityMod(a.Dexterity)
	if value > 15 {
		value = 15
	}
	return value + 2
}

func barbarianAC(a AbilityScores) int {
	return 10 + abilityMod(a.Dexterity) + abilityMod(a.Constitution)
}

func monkAC(a AbilityScores) int {
	return 10 + abilityMod(a.Dexterity) + abilityMod(a.Wisdom)
}

func abilityMod(score int) int { return (score - 10) / 2 }
