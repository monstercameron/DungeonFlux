package combat

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

// EnemyResult records the deterministic thrall action and its measured impact
// timing. Path contains the cells traversed before the slam.
type EnemyResult struct {
	TargetSeat int
	Path       []Cell
	Outcome    rules.AttackOutcome
	ContactMS  int
}

// EnemyTurn advances the thrall's turn. It picks the nearest living PC,
// breaking distance ties by seat, walks at most four cells, and slams when it
// reaches an adjacent cell. The thrall starts at the combat origin because
// the current child-machine state stores its position as the spawn cell.
func (s *State) EnemyTurn(source *dice.Roller, contactMS int) (EnemyResult, error) {
	if s == nil || source == nil {
		return EnemyResult{}, errors.New("combat state or dice source is nil")
	}
	if s.Phase != EnemyTurn {
		return EnemyResult{}, errors.New("enemy turn is not active")
	}
	if contactMS < 0 {
		return EnemyResult{}, errors.New("contact time cannot be negative")
	}
	target, path, ok := s.enemyTarget()
	result := EnemyResult{ContactMS: contactMS}
	if !ok {
		return result, nil
	}
	result.TargetSeat = target.Seat
	result.Path = path
	if len(path) > 0 && !adjacent(path[len(path)-1], target.Position) {
		return result, nil
	}
	outcome, err := rules.Slam(source, "slam-"+target.ID, target.ID, target.AC, target.HP, 0)
	if err != nil {
		return EnemyResult{}, err
	}
	result.Outcome = outcome
	if outcome.Hit && outcome.Total > 0 {
		updated := rules.CreatureState{ID: target.ID, HP: target.HP, MaxHP: target.MaxHP, Conditions: append([]rules.Condition(nil), target.Conditions...)}
		rules.ApplyDamage(&updated, outcome.Total)
		target.HP = updated.HP
		target.Conditions = updated.Conditions
		if err := s.SetParticipant(target); err != nil {
			return EnemyResult{}, err
		}
	}
	return result, nil
}

func (s State) enemyTarget() (Participant, []Cell, bool) {
	origin := s.ThrallCell()
	var chosen Participant
	var chosenPath []Cell
	found := false
	for _, pc := range s.PCs {
		if pc.IsDown() || !s.Grid.IsWalkable(pc.Position) {
			continue
		}
		path, ok := shortestPath(s.Grid, origin, pc.Position, s.Grid.Cols*s.Grid.Rows)
		if !ok {
			continue
		}
		if !found || len(path) < len(chosenPath) || (len(path) == len(chosenPath) && pc.Seat < chosen.Seat) {
			chosen, chosenPath, found = pc, path, true
		}
	}
	if !found {
		return Participant{}, nil, false
	}
	approach := s.enemyApproach(origin, chosen)
	if len(approach) > maxEnemyMove {
		approach = approach[:maxEnemyMove]
	}
	return chosen, approach, true
}

func (s State) enemyApproach(origin Cell, target Participant) []Cell {
	if adjacent(origin, target.Position) {
		return nil
	}
	var best []Cell
	for _, goal := range neighbors(target.Position) {
		path, ok := shortestPath(s.Grid, origin, goal, s.Grid.Cols*s.Grid.Rows)
		if !ok || (best != nil && len(path) >= len(best)) {
			continue
		}
		best = path
	}
	return best
}

const maxEnemyMove = 4
