package phone

import (
	"sort"
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// SheetSnapshot is the private player sheet projected from a phone view.
type SheetSnapshot struct {
	Name               string
	Class              string
	Species            string
	Gender             string
	PortraitURL        string
	Hook               string
	PersuasionModifier int32
	HP                 int32
	HPMax              int32
	AC                 int32
	Level              int32
	Abilities          []StatValue
	SaveProficiencies  []string
	SkillProficiencies map[string]string
	AttackName         string
	AttackDice         string
	AttackDamageType   string
	AttackBonus        int32
	Conditions         []string
	StatusText         string
	Locale             string
	Equipment          []SheetEquipmentItem
}

// SheetEquipmentItem is one piece of starting gear shown on the inventory tab.
type SheetEquipmentItem struct {
	Name        string
	Description string
	Slot        string
	Worn        bool
}

// SheetAction is a display-only action available from the character sheet.
type SheetAction struct {
	ID      string
	Label   string
	Detail  string
	Icon    string
	Enabled bool
}

// SheetActions returns the class's demo actions in stable display order.
func SheetActions(className string) []SheetAction {
	rows := []SheetAction{
		sheetAttackAction(className),
		{ID: "help", Label: "Help Ally", Detail: "Give an ally advantage", Icon: "+", Enabled: true},
		{ID: "dash", Label: "Dash", Detail: "Move twice this turn", Icon: "↗", Enabled: true},
	}
	switch strings.ToLower(strings.TrimSpace(className)) {
	case "rogue":
		rows[1] = SheetAction{ID: "sneak_attack", Label: "Sneak Attack", Detail: "+1d6 when you have the opening", Icon: "◇", Enabled: true}
	case "paladin":
		rows[1] = SheetAction{ID: "lay_on_hands", Label: "Lay on Hands", Detail: "Restore up to 5 HP", Icon: "+", Enabled: true}
	case "bard":
		rows[1] = SheetAction{ID: "inspiration", Label: "Bardic Inspiration", Detail: "Inspire an ally · d6", Icon: "♫", Enabled: true}
	case "cleric":
		rows[1] = SheetAction{ID: "spellcasting", Label: "Spellcasting", Detail: "Channel divine power", Icon: "✦", Enabled: true}
	}
	return rows
}

type sheetAttack struct {
	name, dice, damageType string
	bonus                  int32
}

func sheetAttackForClass(className string) sheetAttack {
	switch strings.ToLower(strings.TrimSpace(className)) {
	case "barbarian":
		return sheetAttack{name: "Greataxe", dice: "1d12+3", damageType: "slashing", bonus: 5}
	case "bard":
		return sheetAttack{name: "Dagger", dice: "1d4+2", damageType: "piercing", bonus: 4}
	case "cleric":
		return sheetAttack{name: "Mace", dice: "1d6", damageType: "bludgeoning", bonus: 2}
	case "druid":
		return sheetAttack{name: "Scimitar", dice: "1d6+3", damageType: "slashing", bonus: 5}
	case "fighter", "paladin":
		return sheetAttack{name: "Longsword", dice: "1d8+3", damageType: "slashing", bonus: 5}
	case "monk":
		return sheetAttack{name: "Quarterstaff", dice: "1d6+3", damageType: "bludgeoning", bonus: 5}
	case "ranger":
		return sheetAttack{name: "Longbow", dice: "1d8+3", damageType: "piercing", bonus: 5}
	case "rogue":
		return sheetAttack{name: "Shortsword", dice: "1d6+3", damageType: "piercing", bonus: 5}
	case "sorcerer", "wizard":
		return sheetAttack{name: "Dagger", dice: "1d4+3", damageType: "piercing", bonus: 5}
	case "warlock":
		return sheetAttack{name: "Light Crossbow", dice: "1d8+3", damageType: "piercing", bonus: 5}
	default:
		return sheetAttack{name: "Attack"}
	}
}

func sheetAttackAction(className string) SheetAction {
	attack := sheetAttackForClass(className)
	detail := "Use your equipped weapon"
	if attack.dice != "" {
		detail = "+" + strconv.Itoa(int(attack.bonus)) + " to hit · " + attack.dice + " " + attack.damageType
	}
	return SheetAction{ID: "attack", Label: attack.name, Detail: detail, Icon: "⚔", Enabled: true}
}

func sheetAbilityValues(values []int32) []StatValue {
	names := [...]string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	if len(values) == 0 {
		return nil
	}
	count := len(values)
	if count > len(names) {
		count = len(names)
	}
	stats := make([]StatValue, 0, count)
	for i := 0; i < count; i++ {
		score := values[i]
		stats = append(stats, StatValue{Name: names[i], Score: score, Modifier: abilityModifier(score)})
	}
	return stats
}

func abilityModifier(score int32) int32 {
	delta := score - 10
	if delta < 0 {
		return -((-delta + 1) / 2)
	}
	return delta / 2
}

// SheetHPPercent returns a clamped percentage for the hit-point meter.
func SheetHPPercent(hp, max int32) int32 {
	if max <= 0 {
		return 0
	}
	if hp <= 0 {
		return 0
	}
	if hp >= max {
		return 100
	}
	return hp * 100 / max
}

// SheetHPClass returns the visual severity class for a hit-point meter.
func SheetHPClass(hp, max int32) string {
	if max <= 0 {
		return "df-phone-hp-unknown"
	}
	if hp <= 0 {
		return "df-phone-hp-down"
	}
	if hp*2 <= max {
		return "df-phone-hp-critical"
	}
	return "df-phone-hp-ready"
}

// SheetModel stores the latest server-authoritative player sheet.
type SheetModel struct {
	state SheetSnapshot
}

// Locale returns the render locale settled from the latest server view.
func (m *SheetModel) Locale() string {
	if m == nil || m.state.Locale == "" {
		return "en"
	}
	return m.state.Locale
}

// NewSheetModel creates an empty player sheet.
func NewSheetModel() *SheetModel { return &SheetModel{} }

// Snapshot returns a copy so callers cannot mutate the model's conditions.
func (m *SheetModel) Snapshot() SheetSnapshot {
	if m == nil {
		return SheetSnapshot{}
	}
	state := m.state
	state.Abilities = append([]StatValue(nil), state.Abilities...)
	state.SaveProficiencies = append([]string(nil), state.SaveProficiencies...)
	state.SkillProficiencies = cloneProficiencies(state.SkillProficiencies)
	state.Conditions = append([]string(nil), state.Conditions...)
	state.Equipment = append([]SheetEquipmentItem(nil), state.Equipment...)
	return state
}

func sheetEquipment(items []*df.EquipmentItem) []SheetEquipmentItem {
	if len(items) == 0 {
		return nil
	}
	out := make([]SheetEquipmentItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		out = append(out, SheetEquipmentItem{Name: item.GetName(), Description: item.GetDescription(), Slot: item.GetSlot(), Worn: item.GetWorn()})
	}
	return out
}

// ApplyScreenState replaces the sheet from the seat-specific server view.
func (m *SheetModel) ApplyScreenState(state *df.ScreenState) SheetSnapshot {
	if m == nil {
		return SheetSnapshot{}
	}
	phone := state.GetPhone()
	if phone == nil {
		return m.Snapshot()
	}
	character := phone.GetCharacter()
	m.state = SheetSnapshot{StatusText: phone.GetStatusText(), Locale: phoneLocale(phone)}
	if character != nil {
		m.state.Name = character.GetName()
		m.state.Class = character.GetClassName()
		m.state.Species = character.GetSpecies()
		m.state.Gender = character.GetGender()
		m.state.PortraitURL = character.GetPortraitUrl()
		m.state.Hook = character.GetHookText()
		m.state.PersuasionModifier = character.GetPersuasionModifier()
		// The server (RULES-008, domain.BuildStats) computes the real attack
		// line; sheetAttackForClass is only a fallback for older fixtures
		// that carry no build data at all.
		fallback := sheetAttackForClass(m.state.Class)
		m.state.AttackName, m.state.AttackDice, m.state.AttackDamageType, m.state.AttackBonus = fallback.name, fallback.dice, fallback.damageType, fallback.bonus
		if build := character.GetBuild(); build != nil {
			m.state.HP, m.state.HPMax = build.GetHp(), build.GetHpMax()
			m.state.AC = build.GetAc()
			m.state.Level = 1
			m.state.Abilities = sheetAbilityValues(build.GetAbilities())
			m.state.SaveProficiencies = append([]string(nil), build.GetSaveProfs()...)
			m.state.SkillProficiencies = cloneProficiencies(build.GetSkillProfs())
			if build.GetAttackName() != "" {
				m.state.AttackName, m.state.AttackDice, m.state.AttackDamageType, m.state.AttackBonus = build.GetAttackName(), build.GetAttackDice(), build.GetAttackDamageType(), build.GetAttackBonus()
			}
			m.state.Equipment = sheetEquipment(build.GetEquipment())
		}
	}
	if combat := phone.GetCombat(); combat != nil {
		m.state.HP, m.state.HPMax = combat.GetHp(), combat.GetHpMax()
		m.state.Conditions = append([]string(nil), combat.GetStatuses()...)
	}
	return m.Snapshot()
}

func cloneProficiencies(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

// SortedSkillProficiencies returns skill IDs in stable display order.
func SortedSkillProficiencies(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// Summary returns a compact label suitable for a narrow phone header.
func (m *SheetModel) Summary(locale string) string {
	state := m.Snapshot()
	parts := []string{state.Name, state.Class}
	if state.Name == "" && state.Class == "" {
		return SheetName(locale, "", "")
	}
	return strings.TrimSpace(strings.Join(parts, " · "))
}
