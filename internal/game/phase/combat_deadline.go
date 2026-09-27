package phase

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const combatDeadlineTimer = "combat_cap"

// Cinematics consume the combat budget; they never restart this deadline.
func combatDeadline() domain.StartTimer {
	return domain.StartTimer{Name: combatDeadlineTimer, After: 30e9, Pausable: true, Scope: domain.Scope{Machine: vocab.MachineSession}}
}

func (m *Machine) expireCombatDeadline() (Result, error) {
	var effects []domain.Effect
	if m.combat.Phase != combat.Done {
		end, err := m.combat.ResolveEnd(combat.ReasonCap, 0)
		if err != nil {
			return Result{}, err
		}
		if end.Outcome == combat.Fled {
			effects = combat.FledAudio()
		}
	}
	m.syncCombatSeats()
	return m.transition(eventCliffhanger, effects)
}

func (m *Machine) leaveCombatTimers() []domain.Effect {
	effects := []domain.Effect{domain.CancelTimer{Name: combatDeadlineTimer}}
	if m.killcam.URL != "" {
		effects = append(effects, domain.CancelTimer{Name: m.killcamTimer()})
	}
	m.killcam = domain.KillCamView{}
	m.killcamTime = 0
	return effects
}
