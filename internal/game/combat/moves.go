package combat

import (
	"errors"
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

const maxCombatMove = 6

// MoveResult describes a successful player movement.
type MoveResult struct {
	Seat int
	Path []Cell
}

// AttackResult describes a resolved PC attack and its approach path.
type AttackResult struct {
	Seat    int
	Path    []Cell
	Outcome rules.AttackOutcome
}

// Move moves the active PC to a reachable cell. Movement does not consume the
// attack action, so an attack may follow it in the same turn.
func (s *State) Move(cell Cell) (MoveResult, error) {
	if s == nil {
		return MoveResult{}, errors.New("combat state is nil")
	}
	pc, ok := s.ActiveParticipant()
	if !ok || pc.IsDown() {
		return MoveResult{}, errors.New("no active player turn")
	}
	path, ok := shortestPath(s.Grid, pc.Position, cell, maxCombatMove)
	if !ok {
		return MoveResult{}, fmt.Errorf("cell %v is not reachable", cell)
	}
	pc.Position = cell
	if err := s.SetParticipant(pc); err != nil {
		return MoveResult{}, err
	}
	s.clearPaths()
	s.setAction(pc.ID, "walk", path)
	s.setReachHighlights(pc)
	return MoveResult{Seat: pc.Seat, Path: path}, nil
}

// Attack approaches the thrall and resolves the active PC's weapon attack.
// The attack consumes the turn action and advances the state to rolling.
func (s *State) Attack(source *dice.Roller, target string) (AttackResult, error) {
	if s == nil || source == nil {
		return AttackResult{}, errors.New("combat state or dice source is nil")
	}
	pc, ok := s.ActiveParticipant()
	if !ok || pc.IsDown() {
		return AttackResult{}, errors.New("no active player turn")
	}
	if pc.ActionUsed {
		return AttackResult{}, errors.New("player action already used")
	}
	if target != s.Thrall.ID || s.Thrall.HP <= 0 {
		return AttackResult{}, errors.New("target is not an alive thrall")
	}
	path, destination, ok := s.approach(pc.Position, maxCombatMove)
	if !ok {
		return AttackResult{}, errors.New("thrall is out of attack range")
	}
	pc.Position = destination
	outcome, err := weaponAttack(source, pc, s.Thrall)
	if err != nil {
		return AttackResult{}, err
	}
	if outcome.Hit && s.sneakEligible(pc) {
		sneak, rollErr := source.Roll(6)
		if rollErr != nil {
			return AttackResult{}, rollErr
		}
		outcome.Damage = append(outcome.Damage, rules.DamageRoll{
			Dice: "1d6", Faces: []int{sneak.Face}, Total: sneak.Face, Type: "piercing", Source: "sneak_attack",
		})
		outcome.Total += sneak.Face
		outcome.HPAfter = maxHPAfter(s.Thrall.HP, outcome.Total)
	}
	if outcome.Hit && outcome.Total > 0 {
		rules.ApplyDamage(&s.Thrall, outcome.Total)
	}
	s.clearPaths()
	s.setAction(pc.ID, "attack", path)
	if outcome.Hit {
		anim := "hit"
		if s.Thrall.HP <= 0 {
			anim = "fall"
		}
		s.setAction(s.Thrall.ID, anim, nil)
		s.setContact(1200, 2000)
		s.setCamera("IMPACT", s.Thrall.ID, false, 250)
		s.setShake(12, 250)
	} else {
		s.clearContact()
		s.setCamera("TURN_FOCUS", pc.ID, true, 250)
	}
	s.clearHighlights()
	pc.ActionUsed = true
	if err := s.SetParticipant(pc); err != nil {
		return AttackResult{}, err
	}
	s.Phase = Rolling
	return AttackResult{Seat: pc.Seat, Path: path, Outcome: outcome}, nil
}

// Bell ends the active fight by fleeing through the bell.
func (s *State) Bell() error {
	if s == nil {
		return errors.New("combat state is nil")
	}
	if s.Phase != PCTurn {
		return errors.New("bell is available only during a player turn")
	}
	s.Phase = Done
	s.finishPresentation(false)
	return nil
}

func (s State) approach(start Cell, limit int) ([]Cell, Cell, bool) {
	bestPath := []Cell(nil)
	bestCell := Cell{}
	for _, cell := range neighbors(s.ThrallCell()) {
		path, ok := shortestPath(s.Grid, start, cell, limit)
		if !ok || (bestPath != nil && len(path) >= len(bestPath)) {
			continue
		}
		bestPath, bestCell = path, cell
	}
	return bestPath, bestCell, bestPath != nil
}

// ThrallCell returns the current engine-owned thrall location.
func (s State) ThrallCell() Cell { return s.ThrallPosition }

func weaponAttack(source *dice.Roller, pc Participant, thrall rules.CreatureState) (rules.AttackOutcome, error) {
	notation, bonus, damageType := weapon(pc.Build.Class)
	outcome, err := rules.Attack(source, fmt.Sprintf("attack-%d", pc.Seat), pc.ID, thrall.ID,
		pc.Build.AttackBonus, thrall.AC, notation, bonus, damageType, thrall.HP, 0)
	if err != nil || !outcome.Hit {
		return outcome, err
	}
	return outcome, nil
}

func (s State) sneakEligible(pc Participant) bool {
	if pc.Build.Class != rules.Rogue || (s.TurnNumber == 1 && pc.Seat == 1) {
		return false
	}
	otherSeat := 3 - pc.Seat
	other, ok := s.Participant(otherSeat)
	if !ok || other.IsDown() {
		return false
	}
	return adjacent(other.Position, s.ThrallCell())
}

func weapon(class rules.Class) (string, int, string) {
	switch class {
	case rules.Paladin:
		return "1d8", 3, "slashing"
	case rules.Rogue:
		return "1d6", 3, "piercing"
	case rules.Bard:
		return "1d4", 2, "piercing"
	default:
		return "1d6", 0, "bludgeoning"
	}
}

func shortestPath(grid Grid, start, goal Cell, limit int) ([]Cell, bool) {
	if !grid.IsWalkable(start) || !grid.IsWalkable(goal) || limit < 0 {
		return nil, false
	}
	if start == goal {
		return []Cell{}, true
	}
	type node struct {
		cell Cell
		path []Cell
	}
	queue := []node{{cell: start}}
	visited := map[Cell]bool{start: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if len(current.path) >= limit {
			continue
		}
		for _, next := range neighbors(current.cell) {
			if visited[next] || !grid.IsWalkable(next) {
				continue
			}
			path := append(append([]Cell(nil), current.path...), next)
			if next == goal {
				return path, true
			}
			visited[next] = true
			queue = append(queue, node{cell: next, path: path})
		}
	}
	return nil, false
}

func neighbors(cell Cell) []Cell {
	return []Cell{
		{X: cell.X - 1, Y: cell.Y - 1}, {X: cell.X, Y: cell.Y - 1}, {X: cell.X + 1, Y: cell.Y - 1},
		{X: cell.X - 1, Y: cell.Y}, {X: cell.X + 1, Y: cell.Y},
		{X: cell.X - 1, Y: cell.Y + 1}, {X: cell.X, Y: cell.Y + 1}, {X: cell.X + 1, Y: cell.Y + 1},
	}
}

func adjacent(a, b Cell) bool {
	dx, dy := a.X-b.X, a.Y-b.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= 1 && dy <= 1 && (dx != 0 || dy != 0)
}

func maxHPAfter(before, damage int) int {
	if damage >= before {
		return 0
	}
	return before - damage
}
