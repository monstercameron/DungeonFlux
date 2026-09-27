package cliffhanger

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMachine_Enter_selectsLiveClipAndStartsNarration(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Step(domain.AssetReady{Slot: string(vocab.SlotCliffhangerClip), Asset: asset("live")}); err != nil {
		t.Fatal(err)
	}
	result, err := machine.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if machine.State() != StatePlaying || result.Clip.ID != "live" {
		t.Fatalf("state=%q clip=%q", machine.State(), result.Clip.ID)
	}
	line, ok := result.Effects[0].(domain.StartLine)
	if !ok || line.Role != vocab.RoleCliffhanger || line.UtteranceID != "line" {
		t.Fatalf("unexpected line effect: %#v", result.Effects)
	}
}

func TestMachine_Enter_usesGenericWhenLiveClipIsNotReady(t *testing.T) {
	machine := newMachine(t)
	result, err := machine.Enter()
	if err != nil {
		t.Fatal(err)
	}
	if result.Clip.ID != "generic" {
		t.Fatalf("clip=%q, want generic", result.Clip.ID)
	}
}

func TestMachine_assetFailure_selectsGenericThenStill(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Step(domain.AssetFailed{Slot: string(vocab.SlotCliffhangerClip)}); err != nil {
		t.Fatal(err)
	}
	selected, ok := machine.clip.Asset()
	if !ok || selected.ID != domain.AssetID("generic") {
		t.Fatalf("selected=%#v, want generic", selected)
	}
	if _, err := machine.Step(domain.AssetFailed{Slot: string(vocab.SlotCliffhangerClip)}); err != nil {
		t.Fatal(err)
	}
	selected, ok = machine.clip.Asset()
	if !ok || selected.ID != domain.AssetID("still") {
		t.Fatalf("selected=%#v, want still", selected)
	}
	result, err := machine.Enter()
	if err != nil || result.Clip.ID != "still" {
		t.Fatalf("entry result=%#v err=%v", result, err)
	}
}

func TestMachine_lineFailure_playsCannedAndCannedDoneEnds(t *testing.T) {
	machine := newMachine(t)
	if _, err := machine.Enter(); err != nil {
		t.Fatal(err)
	}
	failed, err := machine.Step(domain.LineFailed{UtteranceID: "line"})
	if err != nil {
		t.Fatal(err)
	}
	canned := failed.Effects[0].(domain.PlayCanned)
	if canned.UtteranceID != "canned" || canned.AssetID != "canned-audio" {
		t.Fatalf("unexpected canned effect: %#v", canned)
	}
	result, err := machine.Step(domain.LineDone{UtteranceID: "canned"})
	if err != nil || !result.EndCard || machine.State() != StateEnd {
		t.Fatalf("end result=%#v err=%v state=%q", result, err, machine.State())
	}
}

func TestMachine_rejectsStaleCompletionAndInvalidConfig(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("accepted incomplete config")
	}
	machine := newMachine(t)
	if _, err := machine.Enter(); err != nil {
		t.Fatal(err)
	}
	if _, err := machine.Step(domain.LineDone{UtteranceID: "stale"}); err == nil {
		t.Fatal("accepted stale line completion")
	}
	if _, err := machine.Enter(); err == nil {
		t.Fatal("entered cliffhanger twice")
	}
}

func newMachine(t *testing.T) Machine {
	t.Helper()
	machine, err := New(Config{
		LiveClip:        asset("live"),
		GenericClip:     asset("generic"),
		AnimatedStill:   asset("still"),
		LineID:          "line",
		CannedLineID:    "canned",
		CannedAssetID:   "canned-audio",
		NarrationInput:  "Midnight.",
		VoiceID:         "dm",
		ClueFoundViaNPC: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return machine
}

func asset(id string) domain.Asset { return domain.Asset{ID: domain.AssetID(id)} }
