package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

func TestMoves_reachableMoveAndAttack(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 4, Rows: 4}
	c.PCs[0].Position = Cell{X: 3, Y: 3}
	c.PCs[1].Position = Cell{X: 1, Y: 0}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	moved, err := s.Move(Cell{X: 2, Y: 2})
	if err != nil || len(moved.Path) != 1 {
		t.Fatalf("move = %#v, %v", moved, err)
	}
	diceSource := dice.New(nil)
	if err := diceSource.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	attack, err := s.Attack(diceSource, "thrall")
	if err != nil || !attack.Outcome.Crit || s.Thrall.HP >= 12 {
		t.Fatalf("attack = %#v, %v; hp=%d", attack, err, s.Thrall.HP)
	}
	if s.Phase != Rolling {
		t.Fatalf("phase = %s", s.Phase)
	}
}

func TestMoves_rejectBlockedOutOfTurnAndBell(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 3, Rows: 3, Walkable: map[Cell]bool{{X: 2, Y: 2}: true}}
	c.PCs[0].Position = Cell{X: 2, Y: 2}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Move(Cell{X: 0, Y: 0}); err == nil {
		t.Fatal("expected blocked movement rejection")
	}
	if err := s.Bell(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != Done {
		t.Fatalf("phase = %s", s.Phase)
	}
	if err := s.EndPlayerTurn(); err == nil {
		t.Fatal("expected terminal turn rejection")
	}
}
