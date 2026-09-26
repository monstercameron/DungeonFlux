package combat

import (
	"errors"
	"fmt"
	"time"
)

const turnTimerDuration time.Duration = 10_000_000_000

// AutoAction is the action selected when a player turn timer expires.
type AutoAction uint8

const (
	// AutoNone indicates that no automatic action is available.
	AutoNone AutoAction = iota
	// AutoAttack selects the active player's attack action.
	AutoAttack
	// AutoEndTurn selects the active player's end-turn action.
	AutoEndTurn
)

// TurnTimer is a pure virtual timer for a combat player turn.
type TurnTimer struct {
	Total     time.Duration
	Remaining time.Duration
	Paused    bool
	Enabled   bool
}

// NewTurnTimer creates an enabled, unstarted ten-second turn timer.
func NewTurnTimer() TurnTimer {
	return TurnTimer{Total: turnTimerDuration, Enabled: true}
}

// Start resets and arms the turn timer for a new player turn.
func (t *TurnTimer) Start() {
	if t == nil {
		return
	}
	t.Total = turnTimerDuration
	t.Remaining = turnTimerDuration
	t.Paused = false
}

// Advance moves virtual time and reports whether the timer fired.
func (t *TurnTimer) Advance(delta time.Duration) (bool, error) {
	if t == nil {
		return false, errors.New("turn timer is nil")
	}
	if delta < 0 {
		return false, fmt.Errorf("turn timer cannot advance by %s", delta)
	}
	if !t.Enabled || t.Paused || t.Remaining <= 0 {
		return false, nil
	}
	t.Remaining -= delta
	if t.Remaining > 0 {
		return false, nil
	}
	t.Remaining = 0
	return true, nil
}

// Pause freezes the timer at its current remaining duration.
func (t *TurnTimer) Pause() {
	if t != nil && t.Enabled && t.Remaining > 0 {
		t.Paused = true
	}
}

// Resume allows a paused timer to continue counting down.
func (t *TurnTimer) Resume() {
	if t != nil {
		t.Paused = false
	}
}

// SetEnabled enables or disables future turn timers. Disabling also clears a
// currently armed timer; the combat cap is intentionally outside this type.
func (t *TurnTimer) SetEnabled(enabled bool) {
	if t == nil {
		return
	}
	t.Enabled = enabled
	if !enabled {
		t.Remaining = 0
		t.Paused = false
	}
}

// TurnTimerAction selects the timer fallback for the active player.
func (s State) TurnTimerAction() (AutoAction, error) {
	if s.Phase != PCTurn {
		return AutoNone, errors.New("turn timer is not active")
	}
	pc, ok := s.ActiveParticipant()
	if !ok || pc.IsDown() || pc.ActionUsed {
		return AutoEndTurn, nil
	}
	if s.Thrall.HP > 0 {
		if _, _, reachable := s.approach(pc.Position, maxCombatMove); reachable {
			return AutoAttack, nil
		}
	}
	return AutoEndTurn, nil
}

// Skip ends an active combat as a host-requested flee. If the thrall is
// already defeated, the existing killing attack remains the combat outcome.
func (s *State) Skip() error {
	if s == nil {
		return errors.New("combat state is nil")
	}
	if s.Phase == Done {
		return nil
	}
	s.Phase = Done
	return nil
}
