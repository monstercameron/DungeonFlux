package input

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/nested"
	"github.com/monstercameron/DungeonFlux/internal/game/phase"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestWalkInput_RejectsIllegalPhaseEvents(t *testing.T) {
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := step(&machine, domain.LineDone{}); err == nil {
		t.Fatal("line_done in lobby was accepted")
	}
	if got := machine.State(); got != vocab.StateLobby {
		t.Fatalf("illegal event changed state to %q", got)
	}
}

func TestWalkInput_DoubleTapPersuadeIsRejected(t *testing.T) {
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := walkToConversation(&machine); err != nil {
		t.Fatal(err)
	}
	move := domain.Act{Seat: 1, Move: vocab.MovePersuade}
	if err := step(&machine, move); err != nil {
		t.Fatal(err)
	}
	if err := step(&machine, move); err == nil {
		t.Fatal("second persuade tap was accepted")
	}
	if got := machine.State(); got != vocab.StateCheck {
		t.Fatalf("double tap changed state to %q", got)
	}
}

func TestWalkInput_PauseRejectsInputUntilResume(t *testing.T) {
	machine, err := phase.New()
	if err != nil {
		t.Fatal(err)
	}
	if err := step(&machine, domain.HostCmd{Cmd: vocab.HostPause}); err != nil {
		t.Fatal(err)
	}
	if err := step(&machine, domain.LineDone{}); err == nil {
		t.Fatal("line_done while paused was accepted")
	}
	if got := machine.State(); got != vocab.StateLobby {
		t.Fatalf("paused input changed state to %q", got)
	}
	if err := step(&machine, domain.HostCmd{Cmd: vocab.HostResume}); err != nil {
		t.Fatal(err)
	}
	if err := step(&machine, domain.HostCmd{Cmd: vocab.HostStart}); err != nil {
		t.Fatal(err)
	}
	if got := machine.State(); got != vocab.StateCreation {
		t.Fatalf("state after resume = %q, want creation", got)
	}
}

func TestWalkInput_PTTLateResultAfterResetIsRejected(t *testing.T) {
	machine, err := nested.NewPTT("u-old")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(nested.PTTEvent{Kind: nested.PTTEventTalkStart, UtteranceID: "u-old"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(nested.PTTEvent{Kind: nested.PTTEventSTTError, UtteranceID: "u-old"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(nested.PTTEvent{Kind: nested.PTTEventReset}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(nested.PTTEvent{Kind: nested.PTTEventTranscribed, UtteranceID: "u-old"}); err == nil {
		t.Fatal("late transcription after reset was accepted")
	}
	if got := machine.State(); got != nested.StateIdle {
		t.Fatalf("late result changed PTT state to %q", got)
	}
}

func TestWalkInput_LateLineAfterCancellationDoesNotCompleteTurn(t *testing.T) {
	turn, err := nested.NewNPCTurn("u-live")
	if err != nil {
		t.Fatal(err)
	}
	turn.Step(nested.NPCEvent{Kind: nested.NPCEventAudioStarted, UtteranceID: "u-live"})
	effects := turn.Step(nested.NPCEvent{Kind: nested.NPCEventInterrupt, UtteranceID: "u-live"})
	if len(effects) != 1 || effects[0].Kind != vocab.EffectSendAudioCancel {
		t.Fatalf("cancel effects = %#v", effects)
	}
	if effects := turn.Step(nested.NPCEvent{Kind: nested.NPCEventLineDone, UtteranceID: "u-live"}); len(effects) != 0 {
		t.Fatalf("late line_done emitted effects %#v", effects)
	}
	if got := turn.State(); got != nested.NPCDone {
		t.Fatalf("late line_done changed state to %q", got)
	}
}

func TestWalkInput_StaleUtteranceReportsIdentity(t *testing.T) {
	machine, err := nested.NewPTT("u-current")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(nested.PTTEvent{Kind: nested.PTTEventTalkStart, UtteranceID: "u-current"}); err != nil {
		t.Fatal(err)
	}
	_, _, err = machine.Step(nested.PTTEvent{Kind: nested.PTTEventTranscribed, UtteranceID: "u-old"})
	var stale *nested.StaleUtteranceError
	if !errors.As(err, &stale) {
		t.Fatalf("error = %v, want stale utterance error", err)
	}
	if stale.Want != "u-current" || stale.Got != "u-old" {
		t.Fatalf("stale identity = %q/%q", stale.Want, stale.Got)
	}
}
