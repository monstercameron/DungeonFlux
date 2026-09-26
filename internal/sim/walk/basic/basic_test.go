package basic

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/sim"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// phaseEngine adapts the top-level phase table to the simulator contract.
// Phase-specific effects are deliberately absent while those phases are stubs.
type phaseEngine struct {
	machine phase.Machine
}

func newPhaseEngine(t *testing.T) *phaseEngine {
	t.Helper()
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	return &phaseEngine{machine: machine}
}

func (e *phaseEngine) Step(envelope domain.Envelope) domain.StepOut {
	_, err := e.machine.Step(envelope.Event)
	if err != nil {
		return domain.StepOut{Ack: &domain.Ack{Reason: err.Error()}}
	}
	return domain.StepOut{Ack: &domain.Ack{Accepted: true}}
}

func (e *phaseEngine) Inspect() domain.Inspect {
	return domain.Inspect{Path: string(e.machine.State())}
}

func send(t *testing.T, driver *sim.Simulator, event domain.Event) {
	t.Helper()
	output := driver.Send(event)
	if output.Ack != nil && !output.Ack.Accepted {
		t.Fatalf("event %T rejected: %#v", event, output.Ack)
	}
}

func walkToExploration(t *testing.T, driver *sim.Simulator) {
	t.Helper()
	send(t, driver, domain.HostCmd{Cmd: vocab.HostStart})
	send(t, driver, domain.PCLocked{Seat: 1})
	send(t, driver, domain.LineDone{})
}

func walkToEnd(t *testing.T, driver *sim.Simulator) {
	t.Helper()
	walkToExploration(t, driver)
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveTalkVell})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MovePersuade})
	send(t, driver, domain.TimerFired{Name: "roll_resolved"})
	send(t, driver, domain.LineDone{})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveLeave})
	send(t, driver, domain.LineDone{UtteranceID: "hook-arrival"})
	send(t, driver, domain.LineDone{UtteranceID: "stranger"})
	send(t, driver, domain.LineDone{})
	send(t, driver, domain.LineDone{})
}

func TestWalkBasic_StubbedHappyPathReachesEnd(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	walkToEnd(t, driver)

	if got := engine.machine.State(); got != vocab.StateEnd {
		t.Fatalf("final phase = %q, want %q", got, vocab.StateEnd)
	}
	result, err := driver.Advance(0)
	if err != nil {
		t.Fatal(err)
	}
	if result.Steps != 12 {
		t.Fatalf("steps = %d, want 12", result.Steps)
	}
}

func TestWalkBasic_SkipEveryStateReachesEnd(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	for _, state := range expectedSkipStates() {
		send(t, driver, domain.HostCmd{Cmd: vocab.HostSkip})
		if got := engine.machine.State(); got != state {
			t.Fatalf("skip phase = %q, want %q", got, state)
		}
	}
}

func TestWalkBasic_PauseBlocksLineUntilResume(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	send(t, driver, domain.HostCmd{Cmd: vocab.HostStart})
	send(t, driver, domain.PCLocked{Seat: 1})
	send(t, driver, domain.HostCmd{Cmd: vocab.HostPause})

	paused := driver.Send(domain.LineDone{})
	if paused.Ack == nil || paused.Ack.Accepted {
		t.Fatal("line_done was accepted while paused")
	}
	if got := engine.machine.State(); got != vocab.StateOpening {
		t.Fatalf("phase during pause = %q, want %q", got, vocab.StateOpening)
	}

	send(t, driver, domain.HostCmd{Cmd: vocab.HostResume})
	send(t, driver, domain.LineDone{})
	if got := engine.machine.State(); got != vocab.StateExploration {
		t.Fatalf("phase after resume = %q, want %q", got, vocab.StateExploration)
	}
}
