package llmexec

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const testClue = "The lamplighter was dragged toward the old bell tower."

func outcomePrompt(t *testing.T, llm *fakes.FakeLLM) string {
	t.Helper()
	if len(llm.TextCalls) != 1 || len(llm.TextCalls[0].Request.Messages) != 1 {
		t.Fatalf("llm calls = %#v, want one single-message call", llm.TextCalls)
	}
	return llm.TextCalls[0].Request.Messages[0].Text
}

func TestOutcome_Execute_promptGatesTheClue(t *testing.T) {
	cases := []struct {
		name      string
		role      vocab.Role
		input     string
		wantClue  bool
		wantInput string
	}{
		{name: "reveal carries the clue", role: vocab.RoleNPCReveal, input: "Please, his family is worried.", wantClue: true, wantInput: "Please, his family is worried."},
		{name: "refuse never carries the clue", role: vocab.RoleNPCRefuse, input: "Tell me or else.", wantInput: "Tell me or else."},
		{name: "empty input presses her", role: vocab.RoleNPCReveal, input: "  ", wantClue: true, wantInput: "(presses her)"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{"Fine, love."}}}}
			NewOutcomeExecutor(llm, testClue).Execute(context.Background(), domain.StartLine{UtteranceID: "reveal", Role: tc.role, Input: tc.input}, domain.Scope{}, &fakes.FakeInbox{})
			prompt := outcomePrompt(t, llm)
			if got := strings.Contains(prompt, testClue); got != tc.wantClue {
				t.Fatalf("prompt contains clue = %v, want %v; prompt %q", got, tc.wantClue, prompt)
			}
			if !strings.Contains(prompt, tc.wantInput) {
				t.Fatalf("prompt %q does not answer the player's line %q", prompt, tc.wantInput)
			}
			if role := llm.TextCalls[0].Request.Meta.Role; role != tc.role {
				t.Fatalf("call role = %q, want %q", role, tc.role)
			}
		})
	}
}

func TestOutcome_Execute_streamsNarrationAndCompletes(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{"Fine. ", "The bell tower."}}}}
	in := &fakes.FakeInbox{}
	NewOutcomeExecutor(llm, testClue).Execute(context.Background(), domain.StartLine{UtteranceID: "reveal", Role: vocab.RoleNPCReveal, Input: "Please."}, domain.Scope{}, in)
	if len(in.Calls) != 4 {
		t.Fatalf("events = %d, want 4 (two deltas, final, line_done)", len(in.Calls))
	}
	final, ok := in.Calls[2].Envelope.Event.(domain.NarrationDelta)
	if !ok || !final.Final || final.TextSoFar != "Fine. The bell tower." || final.Speaker != "Mother Vell" {
		t.Fatalf("final delta = %#v", in.Calls[2].Envelope.Event)
	}
	if done, ok := in.Calls[3].Envelope.Event.(domain.LineDone); !ok || done.UtteranceID != "reveal" {
		t.Fatalf("last event = %#v, want line_done", in.Calls[3].Envelope.Event)
	}
}

func TestOutcome_Execute_failures(t *testing.T) {
	overlong := strings.Repeat("word ", 30)
	cases := []struct {
		name string
		llm  *fakes.FakeLLM
		clue string
		role vocab.Role
		want vocab.ErrKind
	}{
		{name: "model error", llm: &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Err: errors.New("down")}}}, clue: testClue, role: vocab.RoleNPCReveal, want: vocab.ErrUnavailable},
		{name: "overlong reply", llm: &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{overlong}}}}, clue: testClue, role: vocab.RoleNPCRefuse, want: vocab.ErrBadOutput},
		{name: "empty reply", llm: &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{}}}}, clue: testClue, role: vocab.RoleNPCRefuse, want: vocab.ErrBadOutput},
		{name: "reveal without a clue", llm: &fakes.FakeLLM{}, role: vocab.RoleNPCReveal, want: vocab.ErrBadOutput},
		{name: "wrong role", llm: &fakes.FakeLLM{}, clue: testClue, role: vocab.RoleNPCReply, want: vocab.ErrBadOutput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{}
			NewOutcomeExecutor(tc.llm, tc.clue).Execute(context.Background(), domain.StartLine{UtteranceID: "u", Role: tc.role, Input: "Please."}, domain.Scope{}, in)
			if len(in.Calls) == 0 {
				t.Fatal("no events, want line_failed")
			}
			failed, ok := in.Calls[len(in.Calls)-1].Envelope.Event.(domain.LineFailed)
			if !ok || failed.FailureKind != tc.want {
				t.Fatalf("last event = %#v, want line_failed %q", in.Calls[len(in.Calls)-1].Envelope.Event, tc.want)
			}
		})
	}
}

func TestOutcome_Execute_nilExecutorFails(t *testing.T) {
	in := &fakes.FakeInbox{}
	var e *OutcomeExecutor
	e.Execute(context.Background(), domain.StartLine{UtteranceID: "u", Role: vocab.RoleNPCReveal}, domain.Scope{}, in)
	if len(in.Calls) != 1 {
		t.Fatalf("events = %d, want one line_failed", len(in.Calls))
	}
}
