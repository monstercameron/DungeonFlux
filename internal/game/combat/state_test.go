package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func testConfig() Config {
	return Config{PCs: [2]Participant{
		{Seat: 1, ID: "pc-1", HP: 10, MaxHP: 10, AC: 14},
		{Seat: 2, ID: "pc-2", HP: 10, MaxHP: 10, AC: 14},
	}, Thrall: rules.Thrall("thrall")}
}

func TestState_fixedTurnOrder(t *testing.T) {
	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != PCTurn || s.TurnSeat != 1 {
		t.Fatalf("first turn = %s/%d", s.Phase, s.TurnSeat)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != EnemyTurn {
		t.Fatalf("after pc1 = %s", s.Phase)
	}
	if err := s.EndEnemyTurn(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != PCTurn || s.TurnSeat != 2 || s.ThrallTurns != 1 {
		t.Fatalf("after enemy = %s/%d/%d", s.Phase, s.TurnSeat, s.ThrallTurns)
	}
	if err := s.EndPlayerTurn(); err != nil {
		t.Fatal(err)
	}
	if s.Phase != PCTurn || s.TurnSeat != 1 {
		t.Fatalf("after pc2 = %s/%d", s.Phase, s.TurnSeat)
	}
}

func TestState_rejectsInvalidTransitionsAndConfig(t *testing.T) {
	if _, err := NewState(Config{}); err == nil {
		t.Fatal("expected invalid config")
	}
	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EndEnemyTurn(); err == nil {
		t.Fatal("expected inactive enemy turn rejection")
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err == nil {
		t.Fatal("expected duplicate start rejection")
	}
	if _, ok := s.ActiveParticipant(); !ok {
		t.Fatal("expected active participant")
	}
	if _, ok := s.Participant(0); ok {
		t.Fatal("expected invalid seat rejection")
	}
}

func TestState_downParticipantCannotAct(t *testing.T) {
	c := testConfig()
	c.PCs[0].HP = 0
	c.PCs[0].Conditions = []rules.Condition{rules.Down}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	active, ok := s.ActiveParticipant()
	if !ok || !active.IsDown() {
		t.Fatal("expected down active participant")
	}
}
