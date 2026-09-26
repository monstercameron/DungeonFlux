package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestWithCombatMoveUI_survivesReset(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		state := New(domain.OneShot{}, []byte("move-ui"), WithCombatMoveUI(enabled))
		if state.phase.CombatMoveUI() != enabled {
			t.Fatalf("configured %v, got %v", enabled, state.phase.CombatMoveUI())
		}
		state.Step(domain.Envelope{Event: domain.HostCmd{Cmd: vocab.HostReset}})
		if state.path != vocab.StateLobby || state.phase.CombatMoveUI() != enabled {
			t.Fatalf("after reset (%s) configured %v, got %v", state.path, enabled, state.phase.CombatMoveUI())
		}
	}
}
