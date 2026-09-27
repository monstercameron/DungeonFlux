package conversation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestStep_OrdinaryQuestionsBypassInterpretation(t *testing.T) {
	for _, text := range []string{
		"Mother Vell, what happened to the missing lamplighter? Did you see where they went?",
		"Did he leave a message?", "How far away is the river?", "Watch your step.", "¿Dónde está el farolero?",
	} {
		t.Run(text, func(t *testing.T) {
			result, err := Step(State{Seat: 1}, Event{Event: domain.Transcribed{UtteranceID: "question", Text: text}})
			if err != nil {
				t.Fatal(err)
			}
			assertDialogue(t, result, text)
			// A delayed classifier callback must not convert dispatched dialogue
			// into a roll or count the same utterance twice.
			late, err := Step(result.State, Event{Event: domain.Interpreted{UtteranceID: "question", InterpretationKind: "MOVE", Move: vocab.MovePersuade}})
			if err != nil || len(late.Events) != 0 || len(late.Effects) != 0 || late.State.NPCReplies != 1 {
				t.Fatalf("late classifier changed dialogue: %+v %v", late, err)
			}
		})
	}
}

func TestStep_InterpretFailurePreservesAmbiguousDialogue(t *testing.T) {
	for _, text := range []string{"Why did he persuade you?", "I do not want to persuade her.", "He told me to step away.", "Did he come back later?"} {
		t.Run(text, func(t *testing.T) {
			start, err := Step(State{Seat: 1}, Event{Event: domain.Say{Seat: 1, UtteranceID: "question", Text: text}})
			if err != nil || !start.State.UtteranceInFlight {
				t.Fatalf("expected a keyword interpretation: %+v %v", start, err)
			}
			result, err := Step(start.State, Event{Event: domain.InterpretFailed{UtteranceID: "question"}})
			if err != nil {
				t.Fatal(err)
			}
			assertDialogue(t, result, text)
		})
	}
}

func TestStep_InvalidInterpretationFallsBackWithoutInventingActions(t *testing.T) {
	for _, event := range []domain.Interpreted{
		{InterpretationKind: "MOVE", Move: vocab.MoveLeave},
		{InterpretationKind: "MOVE"},
		{InterpretationKind: "unknown"},
		{InterpretationKind: "DIALOGUE"},
	} {
		t.Run(string(event.Move)+event.InterpretationKind, func(t *testing.T) {
			event.UtteranceID = "question"
			text := "Why did he try to convince you?"
			state := State{Seat: 1, ActiveUtteranceID: "question", UtteranceInFlight: true, Transcript: text}
			result, err := Step(state, Event{Event: event})
			if err != nil {
				t.Fatal(err)
			}
			assertDialogue(t, result, text)
		})
	}
}

func TestKeywordMove_ExplicitFallbackOnly(t *testing.T) {
	for _, tc := range []struct {
		text string
		move vocab.MoveID
	}{
		{"I try to persuade her.", vocab.MovePersuade},
		{"Please convince her!", vocab.MovePersuade},
		{"I step away.", vocab.MoveStepAway},
		{"Later.", vocab.MoveStepAway},
		{"Never mind.", vocab.MoveStepAway},
		{"leave", ""}, {"away", ""}, {"step", ""},
		{"Did you convince him?", ""}, {"I don't persuade her", ""},
	} {
		t.Run(tc.text, func(t *testing.T) {
			if got := keywordMove(tc.text); got != tc.move {
				t.Fatalf("move=%q want=%q", got, tc.move)
			}
		})
	}
}

func assertDialogue(t *testing.T, result Result, text string) {
	t.Helper()
	if result.State.UtteranceInFlight || len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("not one completed dialogue dispatch: %+v", result)
	}
	if final, ok := result.Events[0].(domain.UtteranceFinal); !ok || final.CleanText != text {
		t.Fatalf("event=%+v want original dialogue", result.Events[0])
	}
	if line, ok := result.Effects[0].(domain.StartLine); !ok || line.Input != text || line.Role != vocab.RoleNPCReply {
		t.Fatalf("effect=%+v want NPC reply", result.Effects[0])
	}
}
