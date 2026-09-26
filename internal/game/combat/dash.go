package combat

import (
	"errors"
	"fmt"
)

// Dash spends the active PC's action to double its movement for the turn
// (SRD 5.2.1 Dash, ruling R-D8) and walks to cell along the engine path. The
// token plays the "dash" animation at DashStepMS per cell. No attack can
// follow: the caller ends the turn once the walk has played (DashWalkMS).
func (s *State) Dash(cell Cell) (MoveResult, error) {
	if s == nil {
		return MoveResult{}, errors.New("combat state is nil")
	}
	pc, ok := s.ActiveParticipant()
	if !ok || pc.IsDown() {
		return MoveResult{}, errors.New("no active player turn")
	}
	if pc.ActionUsed {
		return MoveResult{}, errors.New("player action already used")
	}
	target, ok := s.reach(pc).Lookup(cell)
	if !ok {
		return MoveResult{}, fmt.Errorf("cell %v is not reachable with a dash", cell)
	}
	pc.Position = cell
	pc.Moved += len(target.Path)
	pc.ActionUsed, pc.Dashed = true, true
	if err := s.SetParticipant(pc); err != nil {
		return MoveResult{}, err
	}
	s.clearPaths()
	s.setAction(pc.ID, "dash", target.Path)
	s.setStepMS(pc.ID, DashStepMS)
	s.clearHighlights()
	return MoveResult{Seat: pc.Seat, Path: target.Path}, nil
}

// DashPending reports whether the active PC has dashed and its turn is only
// waiting for the walk to play before it ends.
func (s State) DashPending() bool {
	pc, ok := s.ActiveParticipant()
	return ok && pc.Dashed
}

// DashWalkMS is how long the renderers take to play a dash of cells, plus a
// short settle, so the turn ends only after the token arrives.
func DashWalkMS(cells int) int64 {
	if cells < 0 {
		cells = 0
	}
	return int64(cells)*DashStepMS + 400
}
