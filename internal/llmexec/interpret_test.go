package llmexec

import (
	"context"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestInterpretExecutor_ExecutesMoveAndRendersContext(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{
		Value: []byte(`{"clean_text":"I persuade her","kind":"MOVE","move_id":"persuade"}`),
	}}}
	in := &fakes.FakeInbox{}
	e := NewInterpretExecutor(InterpretConfig{LLM: llm, Phase: vocab.StateConversation, Glossary: "persuade: convince"})
	e.Execute(context.Background(), domain.Interpret{
		Seat: 1, UtteranceID: "utt-1", Transcript: "  I persuade her  ",
		Moves: []vocab.MoveID{vocab.MovePersuade, vocab.MoveStepAway}, NPCLastLine: "Try me.",
	}, domain.Scope{Key: "conversation"}, in)

	if len(in.Calls) != 1 {
		t.Fatalf("events = %d, want 1", len(in.Calls))
	}
	got, ok := in.Calls[0].Envelope.Event.(domain.Interpreted)
	if !ok || got.UtteranceID != "utt-1" || got.CleanText != "I persuade her" || got.InterpretationKind != "MOVE" || got.Move != vocab.MovePersuade {
		t.Fatalf("event = %#v", in.Calls[0].Envelope.Event)
	}
	if len(llm.JSONCalls) != 1 {
		t.Fatalf("JSON calls = %d, want 1", len(llm.JSONCalls))
	}
	call := llm.JSONCalls[0]
	if call.Request.Meta.Role != vocab.RoleInterpret || call.Request.Meta.Phase != vocab.StateConversation || call.Request.Meta.Seat != 1 || call.Request.Meta.UtteranceID != "utt-1" {
		t.Fatalf("call metadata = %#v", call.Request.Meta)
	}
	if call.Schema.Name != string(vocab.RoleInterpret) || call.Request.Messages[1].Text == "" {
		t.Fatalf("schema or prompt missing: %#v", call)
	}
}

func TestInterpretExecutor_DefaultsOptionalContextAndNilMove(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{
		Value: []byte(`{"clean_text":"hello","kind":"DIALOGUE","move_id":null}`),
	}}}
	in := &fakes.FakeInbox{}
	NewInterpretExecutor(InterpretConfig{LLM: llm}).Execute(context.Background(), domain.Interpret{
		UtteranceID: "utt-2", Transcript: "hello", NPCLastLine: "   ",
	}, domain.Scope{}, in)

	if len(in.Calls) != 1 {
		t.Fatalf("events = %d, want 1", len(in.Calls))
	}
	got := in.Calls[0].Envelope.Event.(domain.Interpreted)
	if got.InterpretationKind != "DIALOGUE" || got.Move != "" {
		t.Fatalf("interpreted = %#v", got)
	}
	prompt := llm.JSONCalls[0].Request.Messages[1].Text
	if prompt == "" {
		t.Fatal("defaulted prompt is empty")
	}
}

func TestInterpretExecutor_ModelAndSchemaFailuresPostInterpretFailed(t *testing.T) {
	cases := []struct {
		name string
		llm  *fakes.FakeLLM
	}{
		{name: "model error", llm: &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Err: errors.New("offline")}}}},
		{name: "schema invalid", llm: &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: []byte(`{"clean_text":"hello","kind":"UNKNOWN","move_id":null}`)}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{}
			NewInterpretExecutor(InterpretConfig{LLM: tc.llm}).Execute(context.Background(), domain.Interpret{UtteranceID: "utt-fail", Transcript: "hello"}, domain.Scope{}, in)
			if len(in.Calls) != 1 {
				t.Fatalf("events = %d, want 1", len(in.Calls))
			}
			got, ok := in.Calls[0].Envelope.Event.(domain.InterpretFailed)
			if !ok || got.UtteranceID != "utt-fail" {
				t.Fatalf("event = %#v", in.Calls[0].Envelope.Event)
			}
		})
	}
}

func TestInterpretExecutor_NilLLMAndCanceledContextPostNothing(t *testing.T) {
	cases := []struct {
		name string
		ctx  context.Context
		llm  ports.LLM
	}{
		{name: "nil llm", ctx: context.Background()},
		{name: "already canceled", ctx: canceledContext(), llm: &fakes.FakeLLM{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{}
			NewInterpretExecutor(InterpretConfig{LLM: tc.llm}).Execute(tc.ctx, domain.Interpret{UtteranceID: "utt-stop", Transcript: "hello"}, domain.Scope{}, in)
			if tc.name == "nil llm" {
				if len(in.Calls) != 1 {
					t.Fatalf("events = %d, want 1", len(in.Calls))
				}
				if _, ok := in.Calls[0].Envelope.Event.(domain.InterpretFailed); !ok {
					t.Fatalf("event = %#v", in.Calls[0].Envelope.Event)
				}
				return
			}
			if len(in.Calls) != 0 {
				t.Fatalf("canceled events = %d, want 0", len(in.Calls))
			}
		})
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
