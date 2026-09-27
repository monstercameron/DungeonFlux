package wire

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/phase/conversation"
	"github.com/monstercameron/DungeonFlux/internal/llmexec"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestFakeInterpret_PreservesQuestionsAndExplicitLegalActions(t *testing.T) {
	for _, tc := range []struct {
		name, text string
		move       vocab.MoveID
	}{
		{"reported question", "Mother Vell, what happened to the missing lamplighter? Did you see where they went?", ""},
		{"keyword question", "Why did he try to persuade you?", ""},
		{"negated action", "I do not want to persuade her.", ""},
		{"quotation", `He said "persuade her" yesterday.`, ""},
		{"departure question", "Did he leave later?", ""},
		{"ordinary speech", "Tell me about the river.", ""},
		{"spanish question", "¿Dónde está el farolero?", ""},
		{"persuade", "persuade", vocab.MovePersuade},
		{"typed action", "I try to persuade her.", vocab.MovePersuade},
		{"named action", "I persuade Mother Vell.", vocab.MovePersuade},
		{"convince", "Please convince her!", vocab.MovePersuade},
		{"step away", "I step away.", vocab.MoveStepAway},
		{"later", "Later.", vocab.MoveStepAway},
		{"never mind", "Never mind.", vocab.MoveStepAway},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := interpretWithFake(t, tc.text, []vocab.MoveID{vocab.MovePersuade, vocab.MoveStepAway})
			kind := "DIALOGUE"
			if tc.move != "" {
				kind = "MOVE"
			}
			if got.CleanText != tc.text || got.Move != tc.move || got.InterpretationKind != kind {
				t.Fatalf("interpretation=%+v, want preserved text, %s %s", got, kind, tc.move)
			}
		})
	}
}

func interpretWithFake(t *testing.T, text string, moves []vocab.MoveID) domain.Interpreted {
	t.Helper()
	in := &recordingInbox{}
	llmexec.NewInterpretExecutor(llmexec.InterpretConfig{LLM: newFakeLLM()}).Execute(t.Context(), domain.Interpret{
		Seat: 1, UtteranceID: "question", Transcript: text, Moves: moves,
		NPCLastLine: "Try to persuade me, or step away.",
	}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events=%+v", in.events)
	}
	got, ok := in.events[0].(domain.Interpreted)
	if !ok {
		t.Fatalf("event=%T, want interpreted", in.events[0])
	}
	return got
}

func TestFakeInterpret_QuestionProducesNPCReplyInsteadOfAct(t *testing.T) {
	text := "Mother Vell, what happened to the missing lamplighter? Did you see where they went?"
	got := interpretWithFake(t, text, []vocab.MoveID{vocab.MovePersuade, vocab.MoveStepAway})
	result, err := conversation.Step(conversation.State{Seat: 1, UtteranceInFlight: true, ActiveUtteranceID: "question"}, conversation.Event{Event: got})
	if err != nil || len(result.Events) != 1 || len(result.Effects) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if final, ok := result.Events[0].(domain.UtteranceFinal); !ok || final.CleanText != text {
		t.Fatalf("event=%+v, want preserved player dialogue", result.Events[0])
	}
	if line, ok := result.Effects[0].(domain.StartLine); !ok || line.Role != vocab.RoleNPCReply || line.Input != text {
		t.Fatalf("effect=%+v, want NPC reply", result.Effects[0])
	}
}

func TestFakeInterpret_DoesNotInventIllegalMovesOrReadSystemInstructions(t *testing.T) {
	got := interpretWithFake(t, "I persuade her.", []vocab.MoveID{vocab.MoveStepAway})
	if got.Move != "" || got.InterpretationKind != "DIALOGUE" {
		t.Fatalf("illegal move=%+v", got)
	}
	for _, messages := range [][]ports.Message{
		nil,
		{{Role: vocab.MsgSystem, Text: "Classify this utterance: persuade. Phase: conversation. Legal moves: persuade. NPC's last line: none"}},
		{{Role: vocab.MsgUser, Text: "Classify this utterance: hello"}},
	} {
		if output := fakeInterpret(messages); output["kind"] != "DIALOGUE" || output["move_id"] != nil {
			t.Fatalf("unrecognized request invented an action: %+v", output)
		}
	}
}
