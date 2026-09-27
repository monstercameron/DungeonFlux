package rules

// EquipmentItem is one piece of starting gear shown on a build's inventory.
// Slot groups the item for display ("weapon", "armor", "shield", "gear",
// "coin"); Worn marks armor and shields the character has equipped.
type EquipmentItem struct {
	Name        string
	Description string
	Slot        string
	Worn        bool
}

// Equipment returns the class's SRD 5.2.1 starting-equipment option A,
// substituting the class's demo attack weapon (attack.go) for the option's
// listed weapon so the inventory never contradicts the build card's attack
// line. It is demo content, not randomized, and every class returns a
// non-empty list.
func Equipment(class Class) []EquipmentItem {
	weapon := templateFor(class).Attack
	items := []EquipmentItem{weaponItem(weapon)}
	items = append(items, classEquipment(class)...)
	items = append(items, EquipmentItem{Name: "Coin pouch", Description: startingGold(class), Slot: "coin"})
	return items
}

func weaponItem(attack WeaponAttack) EquipmentItem {
	name := attack.Name
	if name == "" {
		name = "Dagger"
	}
	return EquipmentItem{Name: name, Description: "Main weapon, " + attack.Dice + " " + attack.DamageType, Slot: "weapon", Worn: true}
}

func classEquipment(class Class) []EquipmentItem {
	switch class {
	case Barbarian:
		return []EquipmentItem{
			{Name: "4 Handaxes", Description: "Thrown backup weapon", Slot: "gear"},
			{Name: "Explorer's Pack", Description: "Rope, rations, bedroll, and tinderbox", Slot: "gear"},
		}
	case Bard:
		return []EquipmentItem{
			{Name: "Leather Armor", Description: "Light armor, AC 11 + Dex", Slot: "armor", Worn: true},
			{Name: "Lute", Description: "Bardic Inspiration focus", Slot: "gear"},
			{Name: "Diplomat's Pack", Description: "Fine clothes, ink, and paper", Slot: "gear"},
		}
	case Cleric:
		return []EquipmentItem{
			{Name: "Chain Shirt", Description: "Medium armor, AC 13 + Dex (max 2)", Slot: "armor", Worn: true},
			{Name: "Shield", Description: "+2 AC", Slot: "shield", Worn: true},
			{Name: "Holy Symbol", Description: "Spellcasting focus", Slot: "gear"},
			{Name: "Priest's Pack", Description: "Blanket, candles, and rations", Slot: "gear"},
		}
	case Druid:
		return []EquipmentItem{
			{Name: "Leather Armor", Description: "Light armor, AC 11 + Dex", Slot: "armor", Worn: true},
			{Name: "Wooden Shield", Description: "+2 AC", Slot: "shield", Worn: true},
			{Name: "Druidic Focus", Description: "Sprig of mistletoe", Slot: "gear"},
			{Name: "Explorer's Pack", Description: "Rope, rations, bedroll, and tinderbox", Slot: "gear"},
		}
	case Fighter:
		return []EquipmentItem{
			{Name: "Chain Mail", Description: "Heavy armor, AC 18", Slot: "armor", Worn: true},
			{Name: "Shield", Description: "+2 AC", Slot: "shield", Worn: true},
			{Name: "Light Crossbow, 20 bolts", Description: "Ranged backup weapon", Slot: "gear"},
		}
	case Monk:
		return []EquipmentItem{
			{Name: "10 Darts", Description: "Thrown backup weapon", Slot: "gear"},
			{Name: "Explorer's Pack", Description: "Rope, rations, bedroll, and tinderbox", Slot: "gear"},
		}
	case Paladin:
		return []EquipmentItem{
			{Name: "Chain Mail", Description: "Heavy armor, AC 18", Slot: "armor", Worn: true},
			{Name: "Shield", Description: "+2 AC", Slot: "shield", Worn: true},
			{Name: "Holy Symbol", Description: "Spellcasting focus", Slot: "gear"},
			{Name: "Priest's Pack", Description: "Blanket, candles, and rations", Slot: "gear"},
		}
	case Ranger:
		return []EquipmentItem{
			{Name: "Leather Armor", Description: "Light armor, AC 11 + Dex", Slot: "armor", Worn: true},
			{Name: "Quiver, 20 arrows", Description: "Ammunition for the longbow", Slot: "gear"},
			{Name: "Explorer's Pack", Description: "Rope, rations, bedroll, and tinderbox", Slot: "gear"},
		}
	case Rogue:
		return []EquipmentItem{
			{Name: "Leather Armor", Description: "Light armor, AC 11 + Dex", Slot: "armor", Worn: true},
			{Name: "Shortbow, 20 arrows", Description: "Ranged backup weapon", Slot: "gear"},
			{Name: "Thieves' Tools", Description: "Locks and traps", Slot: "gear"},
			{Name: "Burglar's Pack", Description: "Rope, ball bearings, and a dark lantern", Slot: "gear"},
		}
	case Sorcerer:
		return []EquipmentItem{
			{Name: "Arcane Focus", Description: "Crystal spellcasting focus", Slot: "gear"},
			{Name: "Component Pouch", Description: "Spell material components", Slot: "gear"},
			{Name: "Dungeoneer's Pack", Description: "Rope, torches, and rations", Slot: "gear"},
		}
	case Warlock:
		return []EquipmentItem{
			{Name: "Arcane Focus", Description: "Rod spellcasting focus", Slot: "gear"},
			{Name: "Component Pouch", Description: "Spell material components", Slot: "gear"},
			{Name: "Scholar's Pack", Description: "Ink, paper, and a book of lore", Slot: "gear"},
		}
	case Wizard:
		return []EquipmentItem{
			{Name: "Spellbook", Description: "Holds the character's known spells", Slot: "gear"},
			{Name: "Component Pouch", Description: "Spell material components", Slot: "gear"},
			{Name: "Scholar's Pack", Description: "Ink, paper, and a book of lore", Slot: "gear"},
		}
	default:
		return []EquipmentItem{{Name: "Traveler's Clothes", Description: "Plain adventuring garb", Slot: "gear"}}
	}
}

func startingGold(class Class) string {
	switch class {
	case Fighter, Paladin:
		return "10 gp"
	case Barbarian, Rogue, Ranger:
		return "12 gp"
	case Bard, Cleric, Druid, Monk:
		return "8 gp"
	case Sorcerer, Warlock, Wizard:
		return "6 gp"
	default:
		return "5 gp"
	}
}
