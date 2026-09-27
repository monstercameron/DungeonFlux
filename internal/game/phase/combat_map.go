package phase

import (
	"errors"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// dashResolvedTimer ends a PC turn once its Dash walk has played (R-D8).
const dashResolvedTimer = "combat_dash_resolved"

// ConfigureCombatMoveUI sets the combat_move_ui flag: with it off, combat
// has no Move or Dash and the phones get no movement map (§0.21.3).
func (m *Machine) ConfigureCombatMoveUI(enabled bool) {
	if m != nil {
		m.moveUIOff = !enabled
	}
}

// CombatMoveUI reports whether the combat movement map is enabled.
func (m Machine) CombatMoveUI() bool { return !m.moveUIOff }

// applyCombatMove handles move{cell} and dash{cell}. It returns false when
// the act is not a movement. Only the active seat may move its own token.
func (m *Machine) applyCombatMove(action domain.Act) ([]domain.Effect, bool, error) {
	if action.Move != vocab.MoveMove && action.Move != vocab.MoveDash {
		return nil, false, nil
	}
	if m.moveUIOff {
		return nil, true, errors.New("combat movement is disabled")
	}
	if action.Seat != 0 && int(action.Seat) != m.combat.TurnSeat {
		return nil, true, errors.New("only the active seat may move")
	}
	cell := combat.Cell{X: action.Cell.C, Y: action.Cell.R}
	if action.Move == vocab.MoveMove {
		_, err := m.combat.Move(cell)
		return nil, true, err
	}
	result, err := m.combat.Dash(cell)
	if err != nil {
		return nil, true, err
	}
	after := time.Duration(combat.DashWalkMS(len(result.Path))) * time.Millisecond
	return []domain.Effect{domain.StartTimer{Name: dashResolvedTimer, After: after, Pausable: true, Scope: domain.Scope{Machine: vocab.MachineSession}}}, true, nil
}

// stepCombatDash ends the dashing PC's turn when its walk has played. It
// returns false for any other event.
func (m *Machine) stepCombatDash(event domain.Event) (Result, bool, error) {
	timer, ok := event.(domain.TimerFired)
	if !ok || timer.Name != dashResolvedTimer {
		return Result{}, false, nil
	}
	if !m.combat.DashPending() {
		return Result{}, true, nil
	}
	effects, err := m.finishCombatTurn()
	return Result{Effects: effects}, true, err
}

// withCombatMoveUI keeps Move enabled while the engine reach has any cell
// (a Dash may still go further after the normal move is spent) and drops it
// when the flag is off. Dash is not a separate menu entry: the phone map
// sends dash{cell} for a dash-only cell, and the engine accepts it while the
// action is unused (R-D8).
func (m Machine) withCombatMoveUI(seat domain.SeatID, moves []domain.MoveView) []domain.MoveView {
	out := make([]domain.MoveView, 0, len(moves))
	for _, item := range moves {
		if item.ID == vocab.MoveAttack && m.combat.TurnSeat == int(seat) && m.combat.DashPending() {
			item.Enabled, item.Reason = false, "You dashed this turn"
		}
		if item.ID == vocab.MoveMove {
			if m.moveUIOff {
				continue
			}
			if m.combat.Phase == combat.PCTurn && m.combat.TurnSeat == int(seat) && len(m.combat.ReachFor(int(seat)).Cells) > 0 {
				item.Enabled, item.Reason = true, ""
			}
		}
		out = append(out, item)
	}
	return out
}

// decorateCombatMap adds each seat's movement map and the token walk paces
// to a Combat view.
func (m Machine) decorateCombatMap(view *domain.View) {
	if view == nil || m.moveUIOff {
		return
	}
	for index := range view.Seats {
		view.Seats[index].CombatMap = m.combatMapFor(view.Seats[index].Seat)
	}
	pace := func(tokens []domain.TokenView) {
		for index := range tokens {
			tokens[index].StepMS = m.combat.Presentation.Tokens[string(tokens[index].ID)].StepMS
		}
	}
	if view.Combat != nil {
		pace(view.Combat.Tokens)
	}
	if view.Battlefield != nil {
		pace(view.Battlefield.Tokens)
	}
}

// combatMapFor projects the engine combat into one seat's top-down map. The
// active seat also gets its reach; the other seat watches read-only.
func (m Machine) combatMapFor(seat domain.SeatID) *domain.CombatMapView {
	state := m.combat
	out := &domain.CombatMapView{Cols: state.Grid.Cols, Rows: state.Grid.Rows, WalkStepMS: combat.WalkStepMS, DashStepMS: combat.DashStepMS}
	for _, cell := range walkableCells(state.Grid) {
		out.Walkable = append(out.Walkable, domain.Cell{C: cell.X, R: cell.Y})
	}
	for _, pc := range state.PCs {
		token := mapToken(state, pc.ID, pc.Position, pc.HP, pc.MaxHP)
		token.Seat, token.Kind, token.Down = domain.SeatID(pc.Seat), combatKind(string(pc.Build.Class), pc.Species), pc.IsDown()
		token.Active = state.Phase == combat.PCTurn && state.TurnSeat == pc.Seat
		if pc.Seat == int(seat) {
			out.Me = token.ID
		}
		out.Tokens = append(out.Tokens, token)
	}
	thrall := mapToken(state, state.Thrall.ID, state.ThrallPosition, state.Thrall.HP, state.Thrall.MaxHP)
	thrall.Kind, thrall.Enemy, thrall.Down = "thrall", true, state.Thrall.HP <= 0
	thrall.Active = state.Phase == combat.EnemyTurn
	out.Tokens = append(out.Tokens, thrall)
	pc, ok := state.Participant(int(seat))
	if !ok {
		return out
	}
	out.MoveLeft = state.MoveLeft(pc.Seat)
	reach := state.ReachFor(pc.Seat)
	out.CanDash = reach.CanDash
	for _, entry := range reach.Cells {
		out.Reach = append(out.Reach, domain.CombatMapReach{Cell: domain.Cell{C: entry.Cell.X, R: entry.Cell.Y}, Path: domainCells(entry.Path), Dash: entry.Dash})
	}
	return out
}

func mapToken(state combat.State, id string, cell combat.Cell, hp, hpMax int) domain.CombatMapToken {
	visual := state.Presentation.Tokens[id]
	return domain.CombatMapToken{
		ID: domain.TokenID(id), Cell: domain.Cell{C: cell.X, R: cell.Y}, HP: hp, HPMax: hpMax,
		Path: domainCells(visual.Path), Anim: visual.Anim, AnimSeq: visual.AnimSeq, StepMS: int(visual.StepMS),
	}
}

func walkableCells(grid combat.Grid) []combat.Cell {
	cells := make([]combat.Cell, 0, grid.Cols*grid.Rows)
	for row := 0; row < grid.Rows; row++ {
		for column := 0; column < grid.Cols; column++ {
			if cell := (combat.Cell{X: column, Y: row}); grid.IsWalkable(cell) {
				cells = append(cells, cell)
			}
		}
	}
	return cells
}
