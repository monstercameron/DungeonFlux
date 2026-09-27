package phase

import (
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// DebugTurn moves a freshly constructed combat fixture to a requested legal
// turn. It uses the combat state transitions so turn presentation and legal
// move highlights stay synchronized with the authoritative engine state.
func (m *Machine) DebugTurn(turn string) error {
	if m == nil {
		return errors.New("phase machine is nil")
	}
	if m.State() != vocab.StateCombat {
		return errors.New("debug turn requires combat")
	}
	switch turn {
	case "pc1":
		if m.combat.Phase != combat.PCTurn || m.combat.TurnSeat != 1 {
			return errors.New("pc1 turn is not available")
		}
		return nil
	case "thrall":
		if m.combat.Phase != combat.PCTurn || m.combat.TurnSeat != 1 {
			return errors.New("thrall turn is not available")
		}
		return m.combat.EndPlayerTurn()
	case "pc2":
		if m.combat.Phase != combat.PCTurn || m.combat.TurnSeat != 1 {
			return errors.New("pc2 turn is not available")
		}
		if err := m.combat.EndPlayerTurn(); err != nil {
			return err
		}
		if err := m.combat.EndEnemyTurn(); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown debug combat turn %q", turn)
	}
}
