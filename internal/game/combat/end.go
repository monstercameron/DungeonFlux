package combat

import (
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

// Outcome identifies how combat terminated.
type Outcome string

const (
	// Slain means the drowned thrall reached zero HP.
	Slain Outcome = "SLAIN"
	// Fled means combat ended without reducing the thrall to zero HP.
	Fled Outcome = "FLED"
)

// EndReason identifies the event that requested combat termination.
type EndReason string

const (
	// ReasonBell is the normal player flee action.
	ReasonBell EndReason = "BELL"
	// ReasonCap is the 30-second combat cap.
	ReasonCap EndReason = "CAP"
	// ReasonSkip is a host skip.
	ReasonSkip EndReason = "SKIP"
	// ReasonHPZero records a normal killing attack.
	ReasonHPZero EndReason = "HP_ZERO"
)

// EndResult is the terminal combat result.
type EndResult struct {
	Outcome     Outcome
	Reason      EndReason
	SlainBySeat int
}

// ResolveEnd applies the terminal outcome and stabilizes Down PCs. A zero-HP
// thrall always produces SLAIN, even when reason is CAP or SKIP.
func (s *State) ResolveEnd(reason EndReason, slainBySeat int) (EndResult, error) {
	if s == nil {
		return EndResult{}, errors.New("combat state is nil")
	}
	if s.Phase == Done {
		return EndResult{}, errors.New("combat is already done")
	}
	if !validEndReason(reason) {
		return EndResult{}, fmt.Errorf("unknown combat end reason %q", reason)
	}
	result := EndResult{Outcome: Fled, Reason: reason}
	if s.Thrall.HP <= 0 {
		result.Outcome = Slain
		if slainBySeat == 1 || slainBySeat == 2 {
			result.SlainBySeat = slainBySeat
		}
	}
	for i := range s.PCs {
		stabilize(&s.PCs[i])
	}
	s.Phase = Done
	s.finishPresentation(result.Outcome == Slain)
	return result, nil
}

// End resolves a terminal event without assigning a killing seat.
func End(s *State, reason EndReason) (EndResult, error) {
	return s.ResolveEnd(reason, 0)
}

func validEndReason(reason EndReason) bool {
	return reason == ReasonBell || reason == ReasonCap || reason == ReasonSkip || reason == ReasonHPZero
}

func stabilize(pc *Participant) {
	creature := rules.CreatureState{ID: pc.ID, HP: pc.HP, MaxHP: pc.MaxHP, AC: pc.AC, Conditions: append([]rules.Condition(nil), pc.Conditions...)}
	rules.EndCombat(&creature, false)
	pc.HP, pc.Conditions = creature.HP, creature.Conditions
}
