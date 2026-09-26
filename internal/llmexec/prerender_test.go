package llmexec

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPrerenderTextExecutor_ExecutePostsOrderedTextSet(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: json.RawMessage(`{"not_found":"The river remembers.","found":"The bell tolls."}`)}}}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewPrerenderTextExecutor(llm)
	e.Execute(t.Context(), domain.PrerenderText{Set: "stranger", Role: vocab.RoleStrangerLines, Variants: 2, Input: "hook and clue"}, domain.Scope{Key: "run"}, in)
	if len(in.Calls) != 1 {
		t.Fatalf("posts=%d", len(in.Calls))
	}
	event, ok := in.Calls[0].Envelope.Event.(domain.PrerenderTextDone)
	if !ok || event.Set != "stranger" || len(event.Texts) != 2 || event.Texts[0] != "The bell tolls." || event.Texts[1] != "The river remembers." {
		t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
	}
	if len(llm.JSONCalls) != 1 || llm.JSONCalls[0].Schema.Name != string(vocab.RoleStrangerLines) {
		t.Fatalf("calls=%#v", llm.JSONCalls)
	}
}

func TestPrerenderTextExecutor_ExecutePostsFailureForInvalidOutput(t *testing.T) {
	tests := []struct {
		name   string
		result fakes.LLMJSONResult
		effect domain.PrerenderText
	}{
		{name: "model error", result: fakes.LLMJSONResult{Err: errors.New("down")}, effect: domain.PrerenderText{Set: "combat", Role: vocab.RoleCombatOutcomes, Variants: 3, Input: "thrall"}},
		{name: "wrong variants", effect: domain.PrerenderText{Set: "cliff", Role: vocab.RoleCliffhanger, Variants: 3, Input: "brief"}},
		{name: "bad schema output", result: fakes.LLMJSONResult{Value: json.RawMessage(`{"found":"only one"}`)}, effect: domain.PrerenderText{Set: "stranger", Role: vocab.RoleStrangerLines, Variants: 2, Input: "brief"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{PostResult: true}
			NewPrerenderTextExecutor(&fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{tc.result}}).Execute(t.Context(), tc.effect, domain.Scope{}, in)
			if len(in.Calls) != 1 {
				t.Fatalf("posts=%d", len(in.Calls))
			}
			if _, ok := in.Calls[0].Envelope.Event.(domain.PrerenderFailed); !ok {
				t.Fatalf("event=%#v", in.Calls[0].Envelope.Event)
			}
		})
	}
}

func TestPrerenderTextExecutor_ExecuteCancellationDoesNotPost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	in := &fakes.FakeInbox{PostResult: true}
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: json.RawMessage(`{"found":"x","not_found":"y"}`)}}}
	NewPrerenderTextExecutor(llm).Execute(ctx, domain.PrerenderText{Set: "stranger", Role: vocab.RoleStrangerLines, Variants: 2, Input: "brief"}, domain.Scope{}, in)
	if len(in.Calls) != 0 {
		t.Fatalf("posts=%d", len(in.Calls))
	}
}

func TestPrerenderFields_RejectsUnknownRoleAndVariantCount(t *testing.T) {
	for _, tc := range []struct {
		name string
		role vocab.Role
		n    int
	}{
		{name: "unknown", role: vocab.RoleOpening, n: 2},
		{name: "wrong count", role: vocab.RoleCombatOutcomes, n: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := prerenderFields(tc.role, tc.n); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPrerenderTextExecutor_NilInboxStillCallsModel(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: json.RawMessage(`{"found_via_npc":"a","found_via_stranger":"b"}`)}}}
	NewPrerenderTextExecutor(llm).Execute(context.Background(), domain.PrerenderText{Set: "cliff", Role: vocab.RoleCliffhanger, Variants: 2, Input: "brief"}, domain.Scope{}, nil)
	if len(llm.JSONCalls) != 1 {
		t.Fatalf("calls=%d", len(llm.JSONCalls))
	}
}

func TestPrerenderTextExecutor_ExecuteRendersStructuredInput(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: json.RawMessage(`{"found_via_npc":"a","found_via_stranger":"b"}`)}}}
	NewPrerenderTextExecutor(llm).Execute(t.Context(), domain.PrerenderText{
		Set: "cliff", Role: vocab.RoleCliffhanger, Variants: 2,
		Input: `{"brief":"a storm","characters":"two heroes"}`,
	}, domain.Scope{}, &fakes.FakeInbox{PostResult: true})
	if len(llm.JSONCalls) != 1 || llm.JSONCalls[0].Request.Messages[1].Text == "" {
		t.Fatalf("calls=%#v", llm.JSONCalls)
	}
	if llm.JSONCalls[0].Request.Messages[1].Text == `{"brief":"a storm","characters":"two heroes"}` {
		t.Fatal("structured input was not rendered")
	}
}

func TestPrerenderTextExecutor_ExecuteRejectsMissingDependenciesAndInput(t *testing.T) {
	tests := []struct {
		name   string
		exec   *PrerenderTextExecutor
		effect domain.PrerenderText
	}{
		{name: "nil executor", effect: domain.PrerenderText{Set: "set", Input: "input", Role: vocab.RoleStrangerLines, Variants: 2}},
		{name: "nil LLM", exec: NewPrerenderTextExecutor(nil), effect: domain.PrerenderText{Set: "set", Input: "input", Role: vocab.RoleStrangerLines, Variants: 2}},
		{name: "empty set", exec: NewPrerenderTextExecutor(&fakes.FakeLLM{}), effect: domain.PrerenderText{Input: "input", Role: vocab.RoleStrangerLines, Variants: 2}},
		{name: "empty input", exec: NewPrerenderTextExecutor(&fakes.FakeLLM{}), effect: domain.PrerenderText{Set: "set", Role: vocab.RoleStrangerLines, Variants: 2}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := &fakes.FakeInbox{PostResult: true}
			exec := tc.exec
			if exec == nil {
				exec = (*PrerenderTextExecutor)(nil)
			}
			exec.Execute(t.Context(), tc.effect, domain.Scope{}, in)
			if len(in.Calls) != 1 {
				t.Fatalf("posts=%d", len(in.Calls))
			}
		})
	}
}

func TestPrerenderPrompt_FallsBackForMalformedJSON(t *testing.T) {
	template, err := prompts.TemplateFor(vocab.RoleCliffhanger)
	if err != nil {
		t.Fatal(err)
	}
	text, err := prerenderPrompt(template, "already rendered input")
	if err != nil || text != "already rendered input" {
		t.Fatalf("text=%q err=%v", text, err)
	}
	if _, err := prerenderPrompt(template, " "); err == nil {
		t.Fatal("expected empty input error")
	}
}

var _ ports.LLM = (*fakes.FakeLLM)(nil)
