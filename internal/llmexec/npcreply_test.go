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

func TestNPCReply_StartLineStreamsNarrationAndCompletes(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{"Well, ", "speak or drink."}}}}
	in := &fakes.FakeInbox{}
	e := NewNPCReplyExecutor(llm)
	e.StartLine(context.Background(), domain.StartLine{UtteranceID: "u-1", Input: "Where is the lamplighter?"}, domain.Scope{Key: "u-1"}, in)

	if len(llm.TextCalls) != 1 || llm.TextCalls[0].Request.Meta.Role != vocab.RoleNPCReply {
		t.Fatalf("llm call = %#v", llm.TextCalls)
	}
	if len(in.Calls) != 2 {
		t.Fatalf("events = %d, want validated narration and completion", len(in.Calls))
	}
	final := in.Calls[0].Envelope.Event.(domain.NarrationDelta)
	if !final.Final || final.Text != "Well, speak or drink." || final.TextSoFar != final.Text || final.Speaker != "Mother Vell" {
		t.Fatalf("validated line = %#v", final)
	}
	if _, ok := in.Calls[1].Envelope.Event.(domain.LineDone); !ok {
		t.Fatalf("last event = %#v", in.Calls[1].Envelope.Event)
	}
}

func TestNPCReply_StartLinePostsLineFailedOnModelError(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Err: errors.New("down")}}}
	in := &fakes.FakeInbox{}
	NewNpcReplyExecutor(llm).StartLine(context.Background(), domain.StartLine{UtteranceID: "u-2", Input: "Question"}, domain.Scope{}, in)

	if len(in.Calls) != 1 {
		t.Fatalf("events = %d, want 1", len(in.Calls))
	}
	got, ok := in.Calls[0].Envelope.Event.(domain.LineFailed)
	if !ok || got.UtteranceID != "u-2" || got.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("failure = %#v", in.Calls[0].Envelope.Event)
	}
}

func TestNPCReply_StartLineRejectsEmptyAndOverlongOutput(t *testing.T) {
	tests := []struct {
		name   string
		chunks []string
	}{
		{name: "empty"},
		{name: "overlong", chunks: []string{"one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty one twenty two twenty three twenty four twenty five twenty six"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: tc.chunks}}}
			in := &fakes.FakeInbox{}
			NewNPCReplyExecutor(llm).StartLine(context.Background(), domain.StartLine{UtteranceID: "u-3", Input: "Question"}, domain.Scope{}, in)
			if len(in.Calls) == 0 {
				t.Fatal("no result event")
			}
			if _, ok := in.Calls[len(in.Calls)-1].Envelope.Event.(domain.LineFailed); !ok {
				t.Fatalf("last event = %#v", in.Calls[len(in.Calls)-1].Envelope.Event)
			}
		})
	}
}

func TestNPCReply_StartLinePreservesCallErrorKind(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Err: &ports.CallError{Kind: vocab.ErrRateLimited}}}}
	in := &fakes.FakeInbox{}
	NewNPCReplyExecutor(llm).StartLine(context.Background(), domain.StartLine{UtteranceID: "u-4", Input: "Question"}, domain.Scope{}, in)
	got := in.Calls[0].Envelope.Event.(domain.LineFailed)
	if got.FailureKind != vocab.ErrRateLimited {
		t.Fatalf("kind = %q", got.FailureKind)
	}
}
