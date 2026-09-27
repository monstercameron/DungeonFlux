package story

import (
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/hook"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/resolution"
	"github.com/monstercameron/DungeonFlux/internal/sim"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type phaseEngine struct {
	machine phase.Machine
	entries map[vocab.StateID]int
}

func newPhaseEngine(t *testing.T) *phaseEngine {
	t.Helper()
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	return &phaseEngine{machine: machine, entries: map[vocab.StateID]int{vocab.StateLobby: 1}}
}

func (e *phaseEngine) Step(envelope domain.Envelope) domain.StepOut {
	result, err := e.machine.Step(envelope.Event)
	e.entries[e.machine.State()]++
	if err != nil {
		return domain.StepOut{Ack: &domain.Ack{Reason: err.Error()}}
	}
	return domain.StepOut{Effects: result.Effects, Ack: &domain.Ack{Accepted: true}}
}

func (e *phaseEngine) Inspect() domain.Inspect {
	return domain.Inspect{Path: string(e.machine.State())}
}

func finishWalk(t *testing.T, engine *phaseEngine, driver *sim.Simulator) {
	t.Helper()
	if got := engine.machine.State(); got != vocab.StateEnd {
		t.Fatalf("final phase = %q, want %q", got, vocab.StateEnd)
	}
	if !withinWalkBudget(driver.Now(), engine.entries) {
		t.Fatalf("walk exceeded budget: virtual time %s, entries %#v", driver.Now(), engine.entries)
	}
}

func send(t *testing.T, driver *sim.Simulator, event domain.Event) {
	t.Helper()
	output := driver.Send(event)
	if output.Ack == nil || !output.Ack.Accepted {
		t.Fatalf("event %T rejected: %#v", event, output.Ack)
	}
	if _, err := driver.Advance(time.Second); err != nil {
		t.Fatal(err)
	}
}

func walkToExploration(t *testing.T, driver *sim.Simulator) {
	t.Helper()
	send(t, driver, domain.HostCmd{Cmd: vocab.HostStart})
	send(t, driver, domain.PCLocked{Seat: 1})
	send(t, driver, domain.LineDone{UtteranceID: "opening"})
}

func finishFromHook(t *testing.T, driver *sim.Simulator) {
	t.Helper()
	send(t, driver, domain.LineDone{UtteranceID: "hook-arrival"})
	send(t, driver, domain.LineDone{UtteranceID: "stranger"})
	send(t, driver, domain.LineDone{UtteranceID: "combat-outcome"})
	send(t, driver, domain.LineDone{UtteranceID: "cliffhanger"})
}

func TestWalkStory_HappyRollEnds(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	walkToExploration(t, driver)
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveTalkVell})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MovePersuade})
	send(t, driver, domain.TimerFired{Name: "roll_resolved"})
	send(t, driver, domain.LineDone{UtteranceID: "reveal"})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveLeave})
	finishFromHook(t, driver)
	finishWalk(t, engine, driver)
}

func TestWalkStory_IdleAfterResolutionStartsHook(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	walkToExploration(t, driver)
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveTalkVell})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MovePersuade})
	send(t, driver, domain.TimerFired{Name: "roll_resolved"})
	send(t, driver, domain.LineDone{UtteranceID: "reveal"})
	if _, err := driver.Advance(15 * time.Second); err != nil {
		t.Fatal(err)
	}
	if got := engine.machine.State(); got != vocab.StateHookEvent {
		t.Fatalf("idle fallback phase = %q, want %q", got, vocab.StateHookEvent)
	}
	finishFromHook(t, driver)
	finishWalk(t, engine, driver)
}

func TestWalkStory_FailedRollRelocatesClue(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	walkToExploration(t, driver)
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveTalkVell})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MovePersuade})
	send(t, driver, domain.TimerFired{Name: "roll_resolved"})
	send(t, driver, domain.LineDone{UtteranceID: "refuse"})
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveLeave})
	finishFromHook(t, driver)
	finishWalk(t, engine, driver)
}

func TestWalkStory_LeaveFirstReachesEnd(t *testing.T) {
	engine := newPhaseEngine(t)
	driver := sim.New(engine, sim.Script{})
	walkToExploration(t, driver)
	send(t, driver, domain.Act{Seat: 1, Move: vocab.MoveLeave})
	finishFromHook(t, driver)
	finishWalk(t, engine, driver)
}

func TestResolutionStory_SuccessAndFailureUseOwnLine(t *testing.T) {
	config := resolution.Config{
		SuccessUtterance: "reveal", FailureUtterance: "refuse",
		SuccessText: "The clue is in the bell tower.", FailureText: "Mother Vell refuses.",
		SuccessCanned: "reveal-canned", FailureCanned: "refuse-canned",
	}
	for _, test := range []struct {
		name    string
		success bool
		active  domain.UtteranceID
		canned  domain.AssetID
	}{
		{name: "success", success: true, active: "reveal", canned: "reveal-canned"},
		{name: "failure", success: false, active: "refuse", canned: "refuse-canned"},
	} {
		t.Run(test.name, func(t *testing.T) {
			machine, err := resolution.New(test.success, config)
			if err != nil {
				t.Fatal(err)
			}
			started, err := machine.Start()
			if err != nil || len(started.Effects) != 1 {
				t.Fatalf("Start() = %#v, error = %v", started, err)
			}
			failed, err := machine.Step(domain.LineFailed{UtteranceID: test.active})
			if err != nil || len(failed.Effects) != 2 {
				t.Fatalf("LineFailed = %#v, error = %v", failed, err)
			}
			fallback, ok := failed.Effects[1].(domain.PlayCanned)
			if !ok || fallback.AssetID != test.canned {
				t.Fatalf("fallback = %#v, want canned %q", failed.Effects[1], test.canned)
			}
			if _, err := machine.Step(domain.LineDone{UtteranceID: test.active + "-canned"}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestHookStory_RejectsStaleLineAndUsesCannedFallback(t *testing.T) {
	machine, err := hook.New(hook.Config{
		ArrivalClip: "arrival", StrangerUtterance: "stranger", CannedUtterance: "stranger-canned",
		StrangerText: "A letter, for you.", CannedLine: "stranger-canned-asset",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.ClipDone{AssetID: "arrival"}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.LineDone{UtteranceID: "stale"}); err == nil {
		t.Fatal("stale stranger completion was accepted")
	}
	result, err := machine.Step(domain.LineFailed{UtteranceID: "stranger"})
	if err != nil || len(result.Effects) != 2 {
		t.Fatalf("LineFailed = %#v, error = %v", result, err)
	}
	if _, err := machine.Step(domain.LineDone{UtteranceID: "stranger-canned"}); err != nil {
		t.Fatal(err)
	}
	if machine.State() != hook.Done {
		t.Fatalf("hook state = %q, want %q", machine.State(), hook.Done)
	}
}

func TestWithinWalkBudget_RejectsTimeAndEntryOverflow(t *testing.T) {
	if withinWalkBudget(maxWalkTime+time.Second, nil) {
		t.Fatal("budget accepted excessive virtual time")
	}
	if withinWalkBudget(0, map[vocab.StateID]int{vocab.StateConversation: 4}) {
		t.Fatal("budget accepted excessive state entries")
	}
}
