package hook

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_ArrivalClipStartsStrangerAndTriggersCombat(t *testing.T) {
	machine := newMachine(t)
	result, err := machine.Start()
	if err != nil || len(result.Effects) != 1 {
		t.Fatalf("start=%#v err=%v", result, err)
	}
	clip, ok := result.Effects[0].(domain.PlayCanned)
	if !ok || clip.AssetID != "arrival" {
		t.Fatalf("clip=%#v", result.Effects[0])
	}
	result, err = machine.Step(domain.ClipDone{AssetID: "arrival"})
	if err != nil || result.State != StrangerLine || len(result.Effects) != 1 {
		t.Fatalf("line=%#v err=%v", result, err)
	}
	line, ok := result.Effects[0].(domain.StartLine)
	if !ok || line.Role != vocab.RoleStrangerLines || line.UtteranceID != "stranger" {
		t.Fatalf("line effect=%#v", result.Effects[0])
	}
	result, err = machine.Step(domain.LineDone{UtteranceID: "stranger"})
	if err != nil || !result.Combat || result.State != Done {
		t.Fatalf("combat=%#v err=%v", result, err)
	}
}

func TestMachine_FailedStrangerLineUsesCannedFallback(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.ClipDone{AssetID: "arrival"}); err != nil {
		t.Fatal(err)
	}
	result, err := machine.Step(domain.LineFailed{UtteranceID: "stranger"})
	if err != nil || len(result.Effects) != 2 {
		t.Fatalf("fallback=%#v err=%v", result, err)
	}
	play, ok := result.Effects[1].(domain.PlayCanned)
	if !ok || play.UtteranceID != "stranger-canned" || play.AssetID != "stranger-fallback" {
		t.Fatalf("fallback effect=%#v", result.Effects[1])
	}
	result, err = machine.Step(domain.LineDone{UtteranceID: "stranger-canned"})
	if err != nil || !result.Combat {
		t.Fatalf("fallback completion=%#v err=%v", result, err)
	}
}

func TestMachine_RejectsWrongClipAndStaleLine(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Step(domain.ClipDone{AssetID: "other"}); err == nil {
		t.Fatal("wrong clip accepted")
	}
	if _, err := machine.Step(domain.ClipDone{AssetID: "arrival"}); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.LineDone{UtteranceID: "other"}); err == nil {
		t.Fatal("stale line accepted")
	}
}

func TestNew_ValidatesConfig(t *testing.T) {
	if _, err := New(Config{StrangerUtterance: "s", CannedUtterance: "c"}); err == nil {
		t.Fatal("missing clip accepted")
	}
	if _, err := New(Config{ArrivalClip: "a", CannedUtterance: "c"}); err == nil {
		t.Fatal("missing stranger ID accepted")
	}
}

func newMachine(t *testing.T) Machine {
	t.Helper()
	machine, err := New(Config{
		ArrivalClip: "arrival", StrangerUtterance: "stranger", StrangerText: "It followed me from the river.",
		CannedUtterance: "stranger-canned", CannedLine: "stranger-fallback", StrangerHook: "river",
	})
	if err != nil {
		t.Fatal(err)
	}
	return machine
}
