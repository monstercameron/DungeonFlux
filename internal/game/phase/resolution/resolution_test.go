package resolution

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_SuccessStartsRevealAndCompletes(t *testing.T) {
	machine := newMachine(t, true)
	result, err := machine.Start()
	if err != nil || machine.State() != Success || len(result.Effects) != 1 {
		t.Fatalf("start=%#v err=%v", result, err)
	}
	line, ok := result.Effects[0].(domain.StartLine)
	if !ok || line.Role != vocab.RoleNPCReveal || line.UtteranceID != "reveal" || line.Input != "The clue is below." {
		t.Fatalf("line=%#v", result.Effects[0])
	}
	result, err = machine.Step(domain.LineDone{UtteranceID: "reveal"})
	if err != nil || result.State != Done || machine.State() != Done {
		t.Fatalf("done=%#v err=%v", result, err)
	}
}

func TestMachine_FailureUsesCannedFallback(t *testing.T) {
	machine := newMachine(t, false)
	if _, err := machine.Start(); err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step(domain.LineFailed{UtteranceID: "refuse"})
	if err != nil || len(result.Effects) != 2 {
		t.Fatalf("fallback=%#v err=%v", result, err)
	}
	if drop, ok := result.Effects[0].(domain.DropLine); !ok || drop.UtteranceID != "refuse" {
		t.Fatalf("drop=%#v", result.Effects[0])
	}
	play, ok := result.Effects[1].(domain.PlayCanned)
	if !ok || play.UtteranceID != "refuse-canned" || play.AssetID != "asset-refuse" {
		t.Fatalf("play=%#v", result.Effects[1])
	}
	result, err = machine.Step(domain.LineDone{UtteranceID: "refuse-canned"})
	if err != nil || result.State != Done {
		t.Fatalf("fallback done=%#v err=%v", result, err)
	}
}

func TestMachine_RejectsStaleAndInvalidLifecycleEvents(t *testing.T) {
	machine := newMachine(t, true)
	if _, err := machine.Step(domain.LineDone{UtteranceID: "reveal"}); err == nil {
		t.Fatal("step before start accepted")
	}
	if _, err := machine.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.LineDone{UtteranceID: "other"}); err == nil {
		t.Fatal("stale line accepted")
	}
	if _, err := machine.Start(); err == nil {
		t.Fatal("line restarted")
	}
}

func TestNew_RequiresBothUtterances(t *testing.T) {
	if _, err := New(true, Config{FailureUtterance: "failure"}); err == nil {
		t.Fatal("missing success utterance accepted")
	}
}

func newMachine(t *testing.T, success bool) Machine {
	t.Helper()
	machine, err := New(success, Config{
		SuccessUtterance: "reveal", FailureUtterance: "refuse",
		SuccessText: "The clue is below.", FailureText: "I cannot help you.",
		SuccessCanned: "asset-reveal", FailureCanned: "asset-refuse",
	})
	if err != nil {
		t.Fatal(err)
	}
	return machine
}
