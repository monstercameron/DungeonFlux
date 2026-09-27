package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/game/rules/dice"
)

func enemyState(t *testing.T, first, second Cell) State {
	t.Helper()
	c := testConfig()
	c.Grid = Grid{Cols: 8, Rows: 8}
	c.PCs[0].Position = first
	c.PCs[1].Position = second
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestEnemyTurn_targetsNearestAndSlams(t *testing.T) {
	s := enemyState(t, Cell{X: 2, Y: 0}, Cell{X: 5, Y: 5})
	r := dice.New([]byte("nearest"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	result, err := s.EnemyTurn(r, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetSeat != 1 || len(result.Path) != 1 || result.ContactMS != 1200 {
		t.Fatalf("result = %+v", result)
	}
	if !result.Outcome.Hit || s.PCs[0].HP >= 10 {
		t.Fatalf("expected deterministic hit and damage: %+v, pc=%+v", result.Outcome, s.PCs[0])
	}
}

func TestEnemyTurn_tieUsesLowestSeatAndMovesFourCells(t *testing.T) {
	s := enemyState(t, Cell{X: 6, Y: 0}, Cell{X: 0, Y: 6})
	r := dice.New([]byte("tie"))
	result, err := s.EnemyTurn(r, 900)
	if err != nil {
		t.Fatal(err)
	}
	if result.TargetSeat != 1 || len(result.Path) != maxEnemyMove || result.Outcome.Hit {
		t.Fatalf("result = %+v", result)
	}
}

func TestEnemyTurn_skipsDownAndRejectsInvalidTurns(t *testing.T) {
	s := enemyState(t, Cell{X: 1, Y: 0}, Cell{X: 2, Y: 0})
	p := s.PCs[0]
	p.HP = 0
	p.Conditions = []rules.Condition{rules.Down}
	if err := s.SetParticipant(p); err != nil {
		t.Fatal(err)
	}
	r := dice.New([]byte("down"))
	if err := r.ForceD20(20); err != nil {
		t.Fatal(err)
	}
	if _, err := s.EnemyTurn(r, 1000); err != nil {
		t.Fatal(err)
	}
	if s.PCs[1].HP == 10 {
		t.Fatal("expected living PC to be selected")
	}
	if _, err := s.EnemyTurn(dice.New([]byte("down")), -1); err == nil {
		t.Fatal("expected negative contact time rejection")
	}
	s.Phase = PCTurn
	if _, err := s.EnemyTurn(dice.New([]byte("down")), 1); err == nil {
		t.Fatal("expected inactive turn rejection")
	}
}
