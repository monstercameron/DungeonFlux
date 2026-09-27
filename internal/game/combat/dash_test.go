package combat

import (
	"testing"
)

// corridorState builds a 14 x 3 open field: pc-1 at (0,1), pc-2 at (1,0),
// the thrall far right at (13,1).
func corridorState(t *testing.T) State {
	t.Helper()
	c := testConfig()
	c.Grid = Grid{Cols: 14, Rows: 3}
	c.PCs[0].Position = Cell{X: 0, Y: 1}
	c.PCs[1].Position = Cell{X: 1, Y: 0}
	c.SpawnCell = Cell{X: 13, Y: 1}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestReachFor_splitsNormalAndDashCells(t *testing.T) {
	s := corridorState(t)
	reach := s.ReachFor(1)
	if reach.MoveLeft != 6 || !reach.CanDash {
		t.Fatalf("reach header = %+v", reach)
	}
	normal, ok := reach.Lookup(Cell{X: 6, Y: 1})
	if !ok || normal.Dash || len(normal.Path) != 6 || normal.Path[5] != (Cell{X: 6, Y: 1}) {
		t.Fatalf("6-cell normal entry = %+v, %v", normal, ok)
	}
	dash, ok := reach.Lookup(Cell{X: 12, Y: 1})
	if !ok || !dash.Dash || len(dash.Path) != 12 {
		t.Fatalf("12-cell dash entry = %+v, %v", dash, ok)
	}
	if _, ok := reach.Lookup(Cell{X: 1, Y: 0}); ok {
		t.Fatal("reach ends on the other PC's cell")
	}
	if _, ok := reach.Lookup(Cell{X: 13, Y: 1}); ok {
		t.Fatal("reach ends on the thrall's cell")
	}
	if len(s.ReachFor(2).Cells) != 0 {
		t.Fatal("the waiting seat has reach")
	}
}

func TestReachFor_pathsGoAroundOccupiedCells(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 3, Rows: 2}
	c.PCs[0].Position = Cell{X: 0, Y: 0}
	c.PCs[1].Position = Cell{X: 1, Y: 0}
	c.SpawnCell = Cell{X: 1, Y: 1}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.ReachFor(1).Lookup(Cell{X: 2, Y: 0}); ok {
		t.Fatal("path crossed a wall of combatants")
	}
	s.Grid.Walkable = map[Cell]bool{{X: 0, Y: 0}: true, {X: 1, Y: 0}: true, {X: 2, Y: 0}: true, {X: 0, Y: 1}: true, {X: 2, Y: 1}: true}
	s.ThrallPosition = Cell{X: 0, Y: 1}
	s.PCs[1].Position = Cell{X: 2, Y: 1}
	entry, ok := s.ReachFor(1).Lookup(Cell{X: 2, Y: 0})
	if !ok || len(entry.Path) != 2 || entry.Path[0] != (Cell{X: 1, Y: 0}) {
		t.Fatalf("path around = %+v, %v", entry, ok)
	}
}

func TestMove_spendsMovementAndKeepsTheAction(t *testing.T) {
	s := corridorState(t)
	result, err := s.Move(Cell{X: 4, Y: 1})
	if err != nil || len(result.Path) != 4 {
		t.Fatalf("move = %+v, %v", result, err)
	}
	pc, _ := s.Participant(1)
	if pc.Moved != 4 || pc.ActionUsed || s.MoveLeft(1) != 2 {
		t.Fatalf("after move pc = %+v left %d", pc, s.MoveLeft(1))
	}
	visual := s.Presentation.Tokens["pc-1"]
	if visual.Anim != "walk" || visual.StepMS != WalkStepMS || len(visual.Path) != 4 {
		t.Fatalf("walk presentation = %+v", visual)
	}
	if _, err := s.Move(Cell{X: 7, Y: 1}); err == nil {
		t.Fatal("moved 3 cells with 2 left")
	}
	reach := s.ReachFor(1)
	if entry, ok := reach.Lookup(Cell{X: 12, Y: 1}); !ok || !entry.Dash {
		t.Fatalf("dash after a move = %+v, %v", entry, ok)
	}
}

func TestDash_doublesMovementUsesActionAndEndsTurn(t *testing.T) {
	s := corridorState(t)
	result, err := s.Dash(Cell{X: 12, Y: 1})
	if err != nil || len(result.Path) != 12 || result.Seat != 1 {
		t.Fatalf("dash = %+v, %v", result, err)
	}
	pc, _ := s.Participant(1)
	if !pc.ActionUsed || !pc.Dashed || pc.Position != (Cell{X: 12, Y: 1}) || !s.DashPending() {
		t.Fatalf("after dash pc = %+v", pc)
	}
	visual := s.Presentation.Tokens["pc-1"]
	if visual.Anim != "dash" || visual.StepMS != DashStepMS || len(visual.Path) != 12 {
		t.Fatalf("dash presentation = %+v", visual)
	}
	if len(s.ReachFor(1).Cells) != 0 || s.MoveLeft(1) != 0 {
		t.Fatal("movement left after a dash")
	}
	if _, err := s.Dash(Cell{X: 11, Y: 1}); err == nil {
		t.Fatal("second dash accepted")
	}
	if _, err := s.Attack(nil, "thrall"); err == nil {
		t.Fatal("attack accepted after a dash")
	}
	if action, _ := s.TurnTimerAction(); action != AutoEndTurn {
		t.Fatalf("timer fallback after dash = %v", action)
	}
	if DashWalkMS(12) != 12*DashStepMS+400 || DashWalkMS(-1) != 400 {
		t.Fatal("dash walk time")
	}
}

func TestDash_rejectsWrongStates(t *testing.T) {
	var nilState *State
	if _, err := nilState.Dash(Cell{}); err == nil {
		t.Fatal("nil state dashed")
	}
	s := corridorState(t)
	if _, err := s.Dash(Cell{X: 13, Y: 2}); err == nil {
		t.Fatal("dashed 13 cells")
	}
	s.Phase = EnemyTurn
	if _, err := s.Dash(Cell{X: 2, Y: 1}); err == nil {
		t.Fatal("dashed outside a player turn")
	}
	s.Phase = PCTurn
	pc, _ := s.Participant(1)
	pc.ActionUsed = true
	_ = s.SetParticipant(pc)
	if _, err := s.Dash(Cell{X: 2, Y: 1}); err == nil {
		t.Fatal("dashed with the action spent")
	}
	if entry, ok := s.ReachFor(1).Lookup(Cell{X: 8, Y: 1}); ok {
		t.Fatalf("dash reach with the action spent = %+v", entry)
	}
}

func TestAttack_approachUsesRemainingMovement(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 10, Rows: 1}
	c.PCs[0].Position = Cell{X: 0}
	c.PCs[1].Position = Cell{X: 9}
	c.SpawnCell = Cell{X: 5}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if action, _ := s.TurnTimerAction(); action != AutoAttack {
		t.Fatalf("fresh turn fallback = %v", action)
	}
	if _, err := s.Move(Cell{X: 1}); err != nil {
		t.Fatal(err)
	}
	pc, _ := s.Participant(1)
	pc.Moved = 6
	_ = s.SetParticipant(pc)
	if action, _ := s.TurnTimerAction(); action != AutoEndTurn {
		t.Fatalf("spent-movement fallback = %v", action)
	}
}

func TestResetAction_clearsMovementEachTurn(t *testing.T) {
	s := corridorState(t)
	if _, err := s.Dash(Cell{X: 8, Y: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndEnemyTurn(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	pc, _ := s.Participant(1)
	if pc.Moved != 0 || pc.Dashed || pc.ActionUsed || s.MoveLeft(1) != 6 {
		t.Fatalf("next turn pc = %+v", pc)
	}
}
