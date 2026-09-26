package rules

// WeaponAttack describes the one level-one attack shown for a demo build.
type WeaponAttack struct {
	Name       string
	Dice       string
	DamageType string
	Bonus      int
}

type classTemplate struct {
	HitDie           int
	Baseline         AbilityScores
	RollOrder        []string
	PrimaryAbilities []string
	AttackAbility    string
	Attack           WeaponAttack
	Skills           []string
	PersuasionNote   string
	AC               func(AbilityScores) int
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
		return classTemplate{12, AbilityScores{17, 10, 14, 8, 12, 14}, rollOrder("dex", "int", "wis", "con"), []string{"str", "con"}, "str", WeaponAttack{"Greataxe", "1d12+3", "slashing", 5}, []string{"animal_handling", "athletics", "intimidation", "nature", "survival", "persuasion"}, "demo-fixed proficiency; not a class skill", barbarianAC}
	case Bard:
		return classTemplate{8, AbilityScores{8, 15, 14, 10, 14, 14}, rollOrder("str", "int", "wis", "con"), []string{"cha", "dex"}, "dex", WeaponAttack{"Dagger", "1d4+2", "piercing", 4}, []string{"persuasion", "performance", "deception", "insight", "religion", "perception"}, "class skill", lightArmorAC}
	case Cleric:
		return classTemplate{8, AbilityScores{10, 12, 13, 9, 17, 14}, rollOrder("dex", "int", "wis", "con"), []string{"wis", "con"}, "str", WeaponAttack{"Mace", "1d6", "bludgeoning", 2}, []string{"insight", "religion", "persuasion", "medicine", "perception"}, "class skill", clericAC}
	case Druid:
		return classTemplate{8, AbilityScores{10, 17, 13, 8, 15, 14}, rollOrder("str", "int", "wis", "con"), []string{"wis", "con"}, "dex", WeaponAttack{"Scimitar", "1d6+3", "slashing", 5}, []string{"animal_handling", "insight", "medicine", "nature", "perception", "religion", "survival", "persuasion"}, "demo-fixed proficiency; not a class skill", lightArmorAC}
	case Fighter:
		return classTemplate{10, AbilityScores{17, 10, 14, 8, 12, 14}, rollOrder("dex", "int", "wis", "con"), []string{"str", "con"}, "str", WeaponAttack{"Longsword", "1d8+3", "slashing", 5}, []string{"athletics", "intimidation", "perception", "survival", "insight", "persuasion"}, "demo-fixed proficiency; not a class skill", heavyArmorAC}
	case Monk:
		return classTemplate{8, AbilityScores{10, 17, 14, 8, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"dex", "wis"}, "dex", WeaponAttack{"Quarterstaff", "1d6+3", "bludgeoning", 5}, []string{"acrobatics", "athletics", "history", "insight", "religion", "stealth", "persuasion"}, "demo-fixed proficiency; not a class skill", monkAC}
	case Paladin:
		return classTemplate{10, AbilityScores{17, 10, 14, 8, 12, 14}, rollOrder("dex", "int", "wis", "con"), []string{"str", "cha"}, "str", WeaponAttack{"Longsword", "1d8+3", "slashing", 5}, []string{"athletics", "intimidation", "persuasion", "insight", "perception"}, "class skill", heavyArmorAC}
	case Ranger:
		return classTemplate{10, AbilityScores{10, 17, 14, 8, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"dex", "wis"}, "dex", WeaponAttack{"Longbow", "1d8+3", "piercing", 5}, []string{"animal_handling", "athletics", "insight", "investigation", "nature", "perception", "stealth", "survival", "persuasion"}, "demo-fixed proficiency; not a class skill", lightArmorAC}
	case Rogue:
		return classTemplate{8, AbilityScores{8, 17, 14, 10, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"dex", "int"}, "dex", WeaponAttack{"Shortsword", "1d6+3", "piercing", 5}, []string{"sleight_of_hand", "stealth", "persuasion", "deception", "investigation", "acrobatics", "perception"}, "class skill; expertise excludes Persuasion", lightArmorAC}
	case Sorcerer:
		return classTemplate{6, AbilityScores{10, 17, 13, 8, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"cha", "con"}, "dex", WeaponAttack{"Dagger", "1d4+3", "piercing", 5}, []string{"arcana", "deception", "insight", "intimidation", "persuasion", "religion"}, "class skill", noArmorAC}
	case Warlock:
		return classTemplate{8, AbilityScores{10, 17, 13, 8, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"cha", "con"}, "dex", WeaponAttack{"Light Crossbow", "1d8+3", "piercing", 5}, []string{"arcana", "deception", "history", "intimidation", "investigation", "nature", "religion", "persuasion"}, "demo-fixed proficiency; not a class skill", lightArmorAC}
	case Wizard:
		return classTemplate{6, AbilityScores{10, 17, 13, 15, 12, 14}, rollOrder("str", "int", "wis", "con"), []string{"int", "dex"}, "dex", WeaponAttack{"Dagger", "1d4+3", "piercing", 5}, []string{"arcana", "history", "insight", "investigation", "medicine", "nature", "religion", "persuasion"}, "demo-fixed proficiency; not a class skill", noArmorAC}
	default:
		return classTemplate{}
	}
}

func rollOrder(values ...string) []string { return values }

func noArmorAC(a AbilityScores) int { return 12 + abilityMod(a.Dexterity) }

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
