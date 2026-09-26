package full

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/combat"
	"github.com/monstercameron/DungeonFlux/internal/game/combat/combatsim"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/sim"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func combatConfig(thrallHP int) combat.Config {
	return combat.Config{
		PCs: [2]combat.Participant{
			{Seat: 1, ID: "pc-1", Build: rules.Build{Class: rules.Paladin, AttackBonus: 5, HP: 10, MaxHP: 10, AC: 14}, Position: combat.Cell{X: 1, Y: 0}, HP: 10, MaxHP: 10, AC: 14},
			{Seat: 2, ID: "pc-2", Build: rules.Build{Class: rules.Rogue, AttackBonus: 5, HP: 10, MaxHP: 10, AC: 14}, Position: combat.Cell{X: 2, Y: 0}, HP: 10, MaxHP: 10, AC: 14},
		},
		Thrall: rules.CreatureState{ID: "thrall", HP: thrallHP, MaxHP: thrallHP, AC: 8},
		Grid:   combat.Grid{Cols: 4, Rows: 4},
	}
}

func newCombat(t *testing.T, hp int) combatsim.Instance {
	t.Helper()
	instance, err := combatsim.New(combatConfig(hp), []byte("walk-full"))
	if err != nil {
		t.Fatal(err)
	}
	return instance
}

func TestWalkFull_CombatOutcomes(t *testing.T) {
	t.Run("forced critical kills before thrall", func(t *testing.T) {
		instance := newCombat(t, 1)
		err := instance.Run(
			combatsim.Action{Kind: combatsim.Start},
			combatsim.Action{Kind: combatsim.ForceD20, Value: 20},
			combatsim.Action{Kind: combatsim.Attack, Target: "thrall"},
		)
		if err != nil {
			t.Fatal(err)
		}
		if !isOutcome(instance.Outcome, combat.Slain, combat.ReasonHPZero) || instance.Outcome.SlainBySeat != 1 {
			t.Fatalf("critical outcome = %#v", instance.Outcome)
		}
	})

	t.Run("bell flees", func(t *testing.T) {
		instance := newCombat(t, 12)
		if err := instance.Run(combatsim.Action{Kind: combatsim.Start}, combatsim.Action{Kind: combatsim.Bell}); err != nil {
			t.Fatal(err)
		}
		if !isOutcome(instance.Outcome, combat.Fled, combat.ReasonBell) {
			t.Fatalf("bell outcome = %#v", instance.Outcome)
		}
	})

	t.Run("cap ends a no-tap fight", func(t *testing.T) {
		instance := newCombat(t, 12)
		if err := instance.Run(combatsim.Action{Kind: combatsim.Start}, combatsim.Action{Kind: combatsim.Advance, Delta: 30 * time.Second}); err != nil {
			t.Fatal(err)
		}
		if !isOutcome(instance.Outcome, combat.Fled, combat.ReasonCap) {
			t.Fatalf("cap outcome = %#v", instance.Outcome)
		}
	})
}

func TestWalkFull_ForcedD20IsConsumedOnce(t *testing.T) {
	instance := newCombat(t, 30)
	if err := instance.Run(combatsim.Action{Kind: combatsim.Start}, combatsim.Action{Kind: combatsim.ForceD20, Value: 20}, combatsim.Action{Kind: combatsim.Attack, Target: "thrall"}); err != nil {
		t.Fatal(err)
	}
	first := instance.State.Thrall.HP
	if instance.State.TurnSeat != 2 {
		t.Fatalf("after first attack active seat = %d, want 2", instance.State.TurnSeat)
	}
	if err := instance.Step(combatsim.Action{Kind: combatsim.Attack, Target: "thrall"}); err != nil {
		t.Fatal(err)
	}
	second := instance.State.Thrall.HP
	if second >= first {
		t.Fatalf("second attack did not resolve: hp %d then %d", first, second)
	}
	if second == 0 {
		t.Fatalf("unexpectedly killed thrall while checking one-shot force: hp %d", second)
	}
}

func TestWalkFull_PauseAndResumeFreezeCombatCap(t *testing.T) {
	instance := newCombat(t, 12)
	if err := instance.Run(combatsim.Action{Kind: combatsim.Start}, combatsim.Action{Kind: combatsim.Pause}, combatsim.Action{Kind: combatsim.Advance, Delta: 30 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if instance.Now != 0 || instance.State.Phase == combat.Done {
		t.Fatalf("paused combat changed: now=%s phase=%s", instance.Now, instance.State.Phase)
	}
	if err := instance.Run(combatsim.Action{Kind: combatsim.Resume}, combatsim.Action{Kind: combatsim.Advance, Delta: 30 * time.Second}); err != nil {
		t.Fatal(err)
	}
	if !isOutcome(instance.Outcome, combat.Fled, combat.ReasonCap) {
		t.Fatalf("resumed outcome = %#v", instance.Outcome)
	}
}

func TestWalkFull_CombatGuards(t *testing.T) {
	instance := newCombat(t, 12)
	if err := instance.Run(combatsim.Action{Kind: combatsim.Start}); err != nil {
		t.Fatal(err)
	}
	if err := instance.Step(combatsim.Action{Kind: combatsim.Move, Cell: combat.Cell{X: 99, Y: 99}}); err == nil {
		t.Fatal("blocked move was accepted")
	}
	if err := instance.Step(combatsim.Action{Kind: combatsim.Skip}); err != nil {
		t.Fatal(err)
	}
	if !isOutcome(instance.Outcome, combat.Fled, combat.ReasonSkip) {
		t.Fatalf("skip outcome = %#v", instance.Outcome)
	}
}

type phaseDriver struct {
	machine phase.Machine
	entries map[vocab.StateID]int
}

func newPhaseDriver(t *testing.T) *phaseDriver {
	t.Helper()
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	return &phaseDriver{machine: machine, entries: map[vocab.StateID]int{vocab.StateLobby: 1}}
}

func (d *phaseDriver) Step(envelope domain.Envelope) domain.StepOut {
	result, err := d.machine.Step(envelope.Event)
	d.entries[d.machine.State()]++
	if err != nil {
		return domain.StepOut{Ack: &domain.Ack{Reason: err.Error()}}
	}
	_ = result
	return domain.StepOut{Ack: &domain.Ack{Accepted: true}}
}

func (d *phaseDriver) Inspect() domain.Inspect {
	return domain.Inspect{Path: string(d.machine.State())}
}

func TestWalkFull_PhaseCombatReturnsToEnd(t *testing.T) {
	driver := newPhaseDriver(t)
	virtual := sim.New(driver, sim.Script{})
	for _, event := range []domain.Event{
		domain.HostCmd{Cmd: vocab.HostStart},
		domain.TimerFired{Name: "creation_timeout"},
		domain.LineDone{},
		domain.Act{Move: vocab.MoveLeave},
		domain.LineDone{UtteranceID: "hook-arrival"},
		domain.HostCmd{Cmd: vocab.HostSkip},
		domain.LineDone{},
		domain.LineDone{},
	} {
		output := virtual.Send(event)
		if output.Ack != nil && !output.Ack.Accepted {
			t.Fatalf("%T rejected: %#v", event, output.Ack)
		}
	}
	if driver.machine.State() != vocab.StateEnd {
		t.Fatalf("final phase = %q", driver.machine.State())
	}
	if !withinBudget(virtual.Now(), driver.entries) {
		t.Fatalf("walk exceeded time budget: %s", virtual.Now())
	}
}
