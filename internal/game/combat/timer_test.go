package combat

import (
	"testing"
	"time"
)

func TestTurnTimer_firesAfterTenSeconds(t *testing.T) {
	timer := NewTurnTimer()
	timer.Start()
	fired, err := timer.Advance(9 * time.Second)
	if err != nil || fired || timer.Remaining != time.Second {
		t.Fatalf("before deadline: fired=%v remaining=%s err=%v", fired, timer.Remaining, err)
	}
	fired, err = timer.Advance(time.Second)
	if err != nil || !fired || timer.Remaining != 0 {
		t.Fatalf("at deadline: fired=%v remaining=%s err=%v", fired, timer.Remaining, err)
	}
	if fired, err = timer.Advance(time.Second); err != nil || fired {
		t.Fatalf("fired timer retriggered: fired=%v err=%v", fired, err)
	}
}

func TestTurnTimer_pauseAndResumePreserveRemaining(t *testing.T) {
	timer := NewTurnTimer()
	timer.Start()
	if _, err := timer.Advance(4 * time.Second); err != nil {
		t.Fatal(err)
	}
	timer.Pause()
	if fired, err := timer.Advance(time.Minute); err != nil || fired || timer.Remaining != 6*time.Second {
		t.Fatalf("paused timer changed: fired=%v remaining=%s err=%v", fired, timer.Remaining, err)
	}
	timer.Resume()
	if fired, err := timer.Advance(6 * time.Second); err != nil || !fired {
		t.Fatalf("resumed timer did not fire: fired=%v err=%v", fired, err)
	}
}

func TestTurnTimer_disabledDoesNotFire(t *testing.T) {
	timer := NewTurnTimer()
	timer.Start()
	timer.SetEnabled(false)
	if fired, err := timer.Advance(time.Minute); err != nil || fired || timer.Remaining != 0 {
		t.Fatalf("disabled timer fired: fired=%v remaining=%s err=%v", fired, timer.Remaining, err)
	}
	timer.SetEnabled(true)
	timer.Start()
	if timer.Remaining != turnTimerDuration {
		t.Fatalf("restart duration = %s", timer.Remaining)
	}
}

func TestTurnTimer_rejectsNegativeAdvanceAndNil(t *testing.T) {
	timer := NewTurnTimer()
	if fired, err := timer.Advance(-time.Second); err == nil || fired {
		t.Fatalf("negative advance: fired=%v err=%v", fired, err)
	}
	var nilTimer *TurnTimer
	if fired, err := nilTimer.Advance(time.Second); err == nil || fired {
		t.Fatalf("nil timer: fired=%v err=%v", fired, err)
	}
}

func TestState_turnTimerActionChoosesAttackOrEndTurn(t *testing.T) {
	c := testConfig()
	c.Grid = Grid{Cols: 3, Rows: 1}
	c.PCs[0].Position = Cell{X: 0}
	c.SpawnCell = Cell{X: 1}
	s, err := NewState(c)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Start(); err != nil {
		t.Fatal(err)
	}
	action, err := s.TurnTimerAction()
	if err != nil || action != AutoAttack {
		t.Fatalf("legal fallback = %v, %v", action, err)
	}
	c.Grid.Walkable = map[Cell]bool{{X: 0}: true}
	s.Grid = c.Grid
	s.ThrallPosition = Cell{X: 2}
	action, err = s.TurnTimerAction()
	if err != nil || action != AutoEndTurn {
		t.Fatalf("illegal fallback = %v, %v", action, err)
	}
	s.Phase = Rolling
	if _, err := s.TurnTimerAction(); err == nil {
		t.Fatal("timer action accepted outside player turn")
	}
}

func TestState_skipEndsCombatAndIsIdempotent(t *testing.T) {
	s, err := NewState(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Skip(); err != nil || s.Phase != Done {
		t.Fatalf("skip = phase %s err %v", s.Phase, err)
	}
	if err := s.Skip(); err != nil || s.Phase != Done {
		t.Fatalf("repeat skip = phase %s err %v", s.Phase, err)
	}
}
