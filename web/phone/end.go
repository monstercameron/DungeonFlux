package phone

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// EndSnapshot is the immutable final state shown on the player's end card.
type EndSnapshot struct {
	Name        string
	Class       string
	PortraitURL string
	Outcome     string
	FinalHP     int32
	FinalHPMax  int32
	FinalStates []string
	Locale      string
}

// EndModel stores the latest server-authoritative end-card state.
type EndModel struct{ state EndSnapshot }

// NewEndModel creates an empty end-card model.
func NewEndModel() *EndModel { return &EndModel{} }

// ApplyScreenState projects the final character and combat state.
func (m *EndModel) ApplyScreenState(state *df.ScreenState) EndSnapshot {
	if m == nil {
		return EndSnapshot{}
	}
	phone := state.GetPhone()
	if phone == nil {
		return m.Snapshot()
	}
	m.state = EndSnapshot{Outcome: phone.GetStatusText(), Locale: phoneLocale(phone)}
	if character := phone.GetCharacter(); character != nil {
		m.state.Name, m.state.Class, m.state.PortraitURL = character.GetName(), character.GetClassName(), character.GetPortraitUrl()
	}
	if combat := phone.GetCombat(); combat != nil {
		m.state.FinalHP, m.state.FinalHPMax = combat.GetHp(), combat.GetHpMax()
		m.state.FinalStates = append([]string(nil), combat.GetStatuses()...)
	}
	return m.Snapshot()
}

// Snapshot returns a copy of the end-card state.
func (m *EndModel) Snapshot() EndSnapshot {
	if m == nil {
		return EndSnapshot{}
	}
	state := m.state
	state.FinalStates = append([]string(nil), state.FinalStates...)
	return state
}

// EndOutcome returns the server's outcome or a stable fallback label.
func EndOutcome(locale, outcome string) string {
	if outcome != "" {
		return outcome
	}
	if locale == "es" {
		return "Aventura completada"
	}
	return "Adventure complete"
}
