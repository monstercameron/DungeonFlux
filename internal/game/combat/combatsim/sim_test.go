package combatsim

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func simConfig() combat.Config {
	return combat.Config{PCs: [2]combat.Participant{
		{Seat: 1, ID: "pc-1", HP: 10, MaxHP: 10, AC: 14, Position: combat.Cell{X: 2, Y: 2}},
		{Seat: 2, ID: "pc-2", HP: 10, MaxHP: 10, AC: 14, Position: combat.Cell{X: 1, Y: 0}},
	}, Thrall: rules.Thrall("thrall"), Grid: combat.Grid{Cols: 4, Rows: 4}}
}

func TestInstance_virtualPauseAndCap(t *testing.T) {
	i, err := New(simConfig(), []byte("cap"))
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Run(Action{Kind: Start}, Action{Kind: Pause}, Action{Kind: Advance, Delta: combatCap}, Action{Kind: Resume}); err != nil {
		t.Fatal(err)
	}
	if i.State.Phase != combat.PCTurn || i.Now != 0 {
		t.Fatalf("paused instance = phase %s, now %s", i.State.Phase, i.Now)
	}
	if err := i.Step(Action{Kind: Advance, Delta: 30 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if i.Outcome.Outcome != combat.Fled || i.Outcome.Reason != combat.ReasonCap {
		t.Fatalf("outcome = %#v", i.Outcome)
	}
}

func TestInstance_forcedCriticalCanSlay(t *testing.T) {
	i, err := New(simConfig(), []byte("critical"))
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Run(Action{Kind: Start}, Action{Kind: ForceD20, Value: 20}, Action{Kind: Attack, Target: "thrall"}); err != nil {
		t.Fatal(err)
	}
	if i.State.Phase != combat.PCTurn || i.State.TurnSeat != 2 {
		t.Fatalf("state after nonlethal attack = %s/%d", i.State.Phase, i.State.TurnSeat)
	}
}

func TestInstance_rejectsInvalidActions(t *testing.T) {
	i, err := New(simConfig(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := i.Step(Action{Kind: Advance, Delta: -time.Second}); err == nil {
		t.Fatal("expected negative time rejection")
	}
	if err := i.Step(Action{Kind: ActionKind("bad")}); err == nil {
		t.Fatal("expected unknown action rejection")
	}
	if err := i.Step(Action{Kind: Start}); err != nil {
		t.Fatal(err)
	}
	if err := i.Step(Action{Kind: Bell}); err != nil {
		t.Fatal(err)
	}
	if i.Outcome.Outcome != combat.Fled {
		t.Fatalf("outcome = %#v", i.Outcome)
	}
}
