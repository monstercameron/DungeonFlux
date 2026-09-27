package nested

import (
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPTT_Step_completesUploadAndTranscription(t *testing.T) {
	machine, err := NewPTT()
	if err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		name  string
		event PTTEvent
		state vocab.StateID
		kind  vocab.EffectKind
	}{
		{"start", PTTEvent{Kind: PTTEventTalkStart, UtteranceID: "u1"}, StateRecording, ""},
		{"end", PTTEvent{Kind: PTTEventTalkEnd, UtteranceID: "u1"}, StateUploading, EffectUpload},
		{"uploaded", PTTEvent{Kind: PTTEventUploadDone, UtteranceID: "u1"}, StateTranscribing, vocab.EffectTranscribe},
		{"transcribed", PTTEvent{Kind: PTTEventTranscribed, UtteranceID: "u1"}, StateDone, ""},
	}
	for _, tc := range steps {
		t.Run(tc.name, func(t *testing.T) {
			_, effects, err := machine.Step(tc.event)
			if err != nil {
				t.Fatal(err)
			}
			if got := machine.State(); got != tc.state {
				t.Fatalf("state = %q, want %q", got, tc.state)
			}
			if tc.kind == "" {
				if len(effects) != 0 {
					t.Fatalf("effects = %#v, want none", effects)
				}
				return
			}
			if len(effects) != 1 || effects[0].Kind != tc.kind || effects[0].UtteranceID != "u1" {
				t.Fatalf("effects = %#v, want %q for u1", effects, tc.kind)
			}
		})
	}
}

func TestPTT_Step_failureReturnsToIdleOnReset(t *testing.T) {
	machine, err := NewPTT("u2")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []PTTEvent{{Kind: PTTEventTalkStart, UtteranceID: "u2"}, {Kind: PTTEventTalkEnd, UtteranceID: "u2"}, {Kind: PTTEventUploadDone, UtteranceID: "u2"}, {Kind: PTTEventSTTError, UtteranceID: "u2"}} {
		if _, _, err := machine.Step(event); err != nil {
			t.Fatal(err)
		}
	}
	if machine.State() != StateFailed {
		t.Fatalf("state = %q, want failed", machine.State())
	}
	if _, effects, err := machine.Step(PTTEvent{Kind: PTTEventReset}); err != nil || len(effects) != 0 {
		t.Fatalf("reset = effects %#v, err %v", effects, err)
	}
	if machine.State() != StateIdle || machine.UtteranceID() != "" {
		t.Fatalf("after reset = %q/%q, want idle/empty", machine.State(), machine.UtteranceID())
	}
}

func TestPTT_Step_rejectsStaleResultAndUnacceptedEvent(t *testing.T) {
	machine, err := NewPTT()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := machine.Step(PTTEvent{Kind: PTTEventTalkStart, UtteranceID: domain.UtteranceID("u3")}); err != nil {
		t.Fatal(err)
	}
	_, _, err = machine.Step(PTTEvent{Kind: PTTEventTranscribed, UtteranceID: "other"})
	var stale *StaleUtteranceError
	if !errors.As(err, &stale) || stale.Want != "u3" || stale.Got != "other" {
		t.Fatalf("stale error = %v", err)
	}
	if _, _, err = machine.Step(PTTEvent{Kind: PTTEventTranscribed, UtteranceID: "u3"}); err == nil {
		t.Fatal("transcription accepted while recording")
	}
	if machine.State() != StateRecording {
		t.Fatalf("state after rejected events = %q, want recording", machine.State())
	}
}
