package phone

import (
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
	Conditions         []string
	StatusText         string
	Locale             string
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
		{ID: "attack", Label: "Attack", Detail: "Use your equipped weapon", Icon: "⚔", Enabled: true},
		{ID: "help", Label: "Help Ally", Detail: "Give an ally advantage", Icon: "+", Enabled: true},
		{ID: "dash", Label: "Dash", Detail: "Move twice this turn", Icon: "↗", Enabled: true},
	}
	switch strings.ToLower(strings.TrimSpace(className)) {
	case "fighter":
		rows[0] = SheetAction{ID: "attack", Label: "Longsword", Detail: "+5 to hit · 1d8+3 slashing", Icon: "⚔", Enabled: true}
	case "rogue":
		rows[0] = SheetAction{ID: "attack", Label: "Shortsword", Detail: "+5 to hit · 1d6+3 piercing", Icon: "⚔", Enabled: true}
		rows[1] = SheetAction{ID: "sneak_attack", Label: "Sneak Attack", Detail: "+1d6 when you have the opening", Icon: "◇", Enabled: true}
	case "paladin":
		rows[0] = SheetAction{ID: "attack", Label: "Longsword", Detail: "+5 to hit · 1d8+3 slashing", Icon: "⚔", Enabled: true}
		rows[1] = SheetAction{ID: "lay_on_hands", Label: "Lay on Hands", Detail: "Restore up to 5 HP", Icon: "+", Enabled: true}
	case "bard":
		rows[0] = SheetAction{ID: "attack", Label: "Dagger", Detail: "+4 to hit · 1d4+2 piercing", Icon: "⚔", Enabled: true}
		rows[1] = SheetAction{ID: "inspiration", Label: "Bardic Inspiration", Detail: "Inspire an ally · d6", Icon: "♪", Enabled: true}
	case "cleric":
		rows[0] = SheetAction{ID: "attack", Label: "Mace", Detail: "+2 to hit · 1d6 bludgeoning", Icon: "⚔", Enabled: true}
		rows[1] = SheetAction{ID: "spellcasting", Label: "Spellcasting", Detail: "Channel divine power", Icon: "✦", Enabled: true}
	}
	return rows
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
	state.Conditions = append([]string(nil), state.Conditions...)
	return state
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
		if build := character.GetBuild(); build != nil {
			m.state.HP, m.state.HPMax = build.GetHp(), build.GetHpMax()
			m.state.AC = build.GetAc()
			m.state.Level = 1
			m.state.Abilities = sheetAbilityValues(build.GetAbilities())
		}
	}
	if combat := phone.GetCombat(); combat != nil {
		m.state.HP, m.state.HPMax = combat.GetHp(), combat.GetHpMax()
		m.state.Conditions = append([]string(nil), combat.GetStatuses()...)
	}
	return m.Snapshot()
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
