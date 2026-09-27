package conversation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestStep_TranscribedRequestsInterpretation(t *testing.T) {
	result, err := Step(State{Seat: 2}, Event{Event: domain.Transcribed{UtteranceID: "u1", Text: "I try to persuade her"}})
	if err != nil {
		t.Fatal(err)
	}
	if !result.State.UtteranceInFlight || len(result.Effects) != 1 {
		t.Fatalf("state/effects = %#v/%#v", result.State, result.Effects)
	}
	interpret, ok := result.Effects[0].(domain.Interpret)
	if !ok || interpret.Seat != 2 || interpret.UtteranceID != "u1" || len(interpret.Moves) != 2 {
		t.Fatalf("interpret effect = %#v", result.Effects[0])
	}
}

func TestStep_InterpretedDialogueDispatchesUtteranceAndLine(t *testing.T) {
	state := State{Seat: 1, UtteranceInFlight: true, ActiveUtteranceID: "u2"}
	result, err := Step(state, Event{Event: domain.Interpreted{UtteranceID: "u2", CleanText: "Where is the bell?", InterpretationKind: InterpretationDialogue}})
	if err != nil || len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	utterance, ok := result.Events[0].(domain.UtteranceFinal)
	if !ok || utterance.CleanText != "Where is the bell?" {
		t.Fatalf("event = %#v", result.Events[0])
	}
	line, ok := result.Effects[0].(domain.StartLine)
	if !ok || line.Role != vocab.RoleNPCReply || line.UtteranceID != "u2" {
		t.Fatalf("line = %#v", result.Effects[0])
	}
}

func TestStep_InterpretedMoveDispatchesAct(t *testing.T) {
	result, err := Step(State{Seat: 1, UtteranceInFlight: true, ActiveUtteranceID: "u3"}, Event{Event: domain.Interpreted{UtteranceID: "u3", InterpretationKind: InterpretationMove, Move: vocab.MovePersuade}})
	if err != nil || len(result.Events) != 1 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	act, ok := result.Events[0].(domain.Act)
	if !ok || act.Seat != 1 || act.Move != vocab.MovePersuade {
		t.Fatalf("event = %#v", result.Events[0])
	}
}

func TestStep_InterpretFailedUsesKeywordFallback(t *testing.T) {
	state := State{Seat: 1}
	transcribed, err := Step(state, Event{Event: domain.Transcribed{UtteranceID: "u4", Text: "Please convince her!"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Step(transcribed.State, Event{Event: domain.InterpretFailed{UtteranceID: "u4"}})
	if err != nil || len(result.Events) != 1 {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if act, ok := result.Events[0].(domain.Act); !ok || act.Move != vocab.MovePersuade {
		t.Fatalf("fallback event = %#v", result.Events[0])
	}
}

func TestStep_IgnoresStaleAndUnsupportedEvents(t *testing.T) {
	state := State{Seat: 1, UtteranceInFlight: true}
	result, err := Step(state, Event{Event: domain.Interpreted{UtteranceID: "old", CleanText: "hi", InterpretationKind: InterpretationDialogue}})
	if err != nil || len(result.Events) != 0 || result.State.UtteranceInFlight != state.UtteranceInFlight {
		t.Fatalf("stale result = %#v, err = %v", result, err)
	}
	result, err = Step(state, Event{Event: domain.LineFirstAudio{UtteranceID: "u5"}})
	if err != nil || !result.State.VoiceBusy {
		t.Fatalf("line start = %#v, err = %v", result, err)
	}
}

func TestStep_RejectsNilEvent(t *testing.T) {
	if _, err := Step(State{}, Event{}); err == nil {
		t.Fatal("nil event accepted")
	}
}

func TestStep_IgnoresInvalidAndCompletedSpeech(t *testing.T) {
	for _, event := range []domain.Event{
		domain.Transcribed{},
		domain.Interpreted{UtteranceID: "u1", CleanText: "hello", InterpretationKind: "unknown"},
		domain.Interpreted{UtteranceID: "u1", CleanText: "", InterpretationKind: InterpretationDialogue},
	} {
		result, err := Step(State{Done: true}, Event{Event: event})
		if err != nil || len(result.Events) != 0 || len(result.Effects) != 0 {
			t.Fatalf("event %#v produced %#v, err %v", event, result, err)
		}
	}
}

func TestStep_KeywordStepAwayAndVoiceLifecycle(t *testing.T) {
	start, err := Step(State{Seat: 2}, Event{Event: domain.Transcribed{UtteranceID: "u5", Text: "step away"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := Step(start.State, Event{Event: domain.InterpretFailed{UtteranceID: "u5"}})
	if err != nil || len(result.Events) != 1 {
		t.Fatalf("step-away fallback = %#v, err %v", result, err)
	}
	if act := result.Events[0].(domain.Act); act.Move != vocab.MoveStepAway || act.Seat != 2 {
		t.Fatalf("act = %#v", act)
	}
	result, err = Step(State{}, Event{Event: domain.TimerFired{Name: "idle_elapsed"}})
	if err != nil || !result.State.IdleElapsed {
		t.Fatalf("idle timer = %#v, err %v", result, err)
	}
	result, err = Step(State{}, Event{Event: domain.LineFirstAudio{UtteranceID: "u6"}})
	if err != nil || !result.State.VoiceBusy {
		t.Fatalf("line first audio = %#v, err %v", result, err)
	}
	result, err = Step(result.State, Event{Event: domain.LineDone{UtteranceID: "u6"}})
	if err != nil || result.State.VoiceBusy {
		t.Fatalf("line done = %#v, err %v", result, err)
	}
}
