package sim

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type testEngine struct {
	path  string
	steps int
}

func (e *testEngine) Step(env domain.Envelope) domain.StepOut {
	e.steps++
	switch env.Event.(type) {
	case domain.HostCmd:
		return domain.StepOut{Effects: []domain.Effect{domain.StartTimer{Name: "done", After: 2 * time.Second}}}
	case domain.TimerFired:
		e.path = string(vocab.StateEnd)
	}
	return domain.StepOut{}
}

func (e *testEngine) Inspect() domain.Inspect { return domain.Inspect{Path: e.path} }

type scriptEngine struct{ steps int }

func (e *scriptEngine) Step(env domain.Envelope) domain.StepOut {
	e.steps++
	return domain.StepOut{Effects: []domain.Effect{domain.StartLine{}}}
}

func (e *scriptEngine) Inspect() domain.Inspect { return domain.Inspect{} }

func TestSimulator_RunUntilEnd_AdvancesTimers(t *testing.T) {
	engine := &testEngine{}
	sim := New(engine, Script{})
	result, err := sim.RunUntilEnd(domain.HostCmd{Cmd: vocab.HostStart}, 5*time.Second)
	if err != nil {
		t.Fatalf("RunUntilEnd() error = %v", err)
	}
	if result.Now != 2*time.Second || result.Steps != 2 {
		t.Fatalf("result = %#v, want 2 seconds and two steps", result)
	}
}

func TestSimulator_ScriptedOutcome_DeliversAfterDelay(t *testing.T) {
	engine := &scriptEngine{}
	sim := New(engine, Script{Outcomes: []Outcome{Success(vocab.EffectStartLine, 3*time.Second, domain.TimerFired{Name: "scripted"})}})
	sim.Send(domain.HostCmd{Cmd: vocab.HostStart})
	if _, err := sim.Advance(2 * time.Second); err != nil {
		t.Fatal(err)
	}
	if engine.steps != 1 {
		t.Fatalf("steps at 2 seconds = %d, want 1", engine.steps)
	}
	result, err := sim.Advance(time.Second)
	if err != nil || result.Steps != 2 {
		t.Fatalf("result = %#v, error = %v", result, err)
	}
}

func TestSimulator_SilenceAndNegativeAdvance(t *testing.T) {
	engine := &testEngine{}
	sim := New(engine, Script{Outcomes: []Outcome{Silence(vocab.EffectStartLine)}})
	sim.Send(domain.HostCmd{Cmd: vocab.HostStart})
	if _, err := sim.Advance(-time.Second); err == nil {
		t.Fatal("Advance() accepted negative duration")
	}
	if _, err := sim.Advance(3 * time.Second); err != nil {
		t.Fatal(err)
	}
	if engine.steps != 2 {
		t.Fatalf("steps = %d, want timer only", engine.steps)
	}
}

func TestSimulator_RunUntilEnd_ReportsLimitAndEmptyQueue(t *testing.T) {
	engine := &testEngine{}
	sim := New(engine, Script{})
	if _, err := sim.RunUntilEnd(domain.HostCmd{Cmd: vocab.HostStart}, time.Second); err == nil {
		t.Fatal("RunUntilEnd() accepted an insufficient limit")
	}
	if _, err := New(engine, Script{}).RunUntilEnd(domain.HostCmd{Cmd: vocab.HostStart}, -time.Second); err == nil {
		t.Fatal("RunUntilEnd() accepted negative limit")
	}
}
