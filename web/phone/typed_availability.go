package phone

import (
	"strconv"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ApplyScreenState gates typed sending from the authoritative phase and turn.
// The draft remains editable while waiting or paused.
func (m *TypedInputModel) ApplyScreenState(state *df.ScreenState) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	locale := phoneLocale(state.GetPhone())
	m.locale, m.state.Locale = locale, locale
	key := ""
	switch {
	case state.GetPaused():
		key = "typed.paused"
	case !strings.EqualFold(state.GetPhase(), "Conversation"):
		key = "typed.unavailable"
	case state.GetPhone().GetPlayerNumber() < 1 || state.GetSpotlightSeat() != strconv.Itoa(int(state.GetPhone().GetPlayerNumber())):
		key = "typed.waiting"
	}
	m.state.BlockedReason = ""
	if key != "" {
		m.state.BlockedReason = T(locale, key, nil)
	}
}
