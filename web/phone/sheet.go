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
	PortraitURL        string
	Hook               string
	PersuasionModifier int32
	HP                 int32
	HPMax              int32
	Conditions         []string
	StatusText         string
	Locale             string
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
		m.state.PortraitURL = character.GetPortraitUrl()
		m.state.Hook = character.GetHookText()
		m.state.PersuasionModifier = character.GetPersuasionModifier()
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
