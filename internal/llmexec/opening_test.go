package llmexec

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type inbox struct{ events []domain.Event }

func (i *inbox) Post(_ context.Context, env domain.Envelope) bool {
	i.events = append(i.events, env.Event)
	return true
}

func TestOpeningExecutor_StreamsDeltas(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{"Rain falls.", " The bell waits."}}}}
	in := &inbox{}
	NewOpeningExecutor(llm).Execute(context.Background(), domain.StartLine{
		Role: vocab.RoleOpening, UtteranceID: "opening-1",
		Input: `{"one_shot":"A missing lamplighter","characters":"Asha and Bram"}`,
	}, domain.Scope{Key: "opening"}, in)
	if len(in.events) != 3 {
		t.Fatalf("events = %d, want 3", len(in.events))
	}
	first := in.events[0].(domain.NarrationDelta)
	if first.Text != "Rain falls." || first.UtteranceID != "opening-1" {
		t.Fatalf("first delta = %#v", first)
	}
	if _, ok := in.events[2].(domain.LineDone); !ok {
		t.Fatalf("last event = %#v, want LineDone", in.events[2])
	}
	if len(llm.TextCalls) != 1 || llm.TextCalls[0].Request.Messages[1].Text == "" {
		t.Fatal("opening prompt was not sent")
	}
}

func TestOpeningExecutor_FailurePostsLineFailed(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Err: errors.New("offline")}}}
	in := &inbox{}
	NewOpeningExecutor(llm).Execute(context.Background(), domain.StartLine{
		Role: vocab.RoleOpening, UtteranceID: "opening-2", Input: "one-shot",
	}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %d, want 1", len(in.events))
	}
	failure, ok := in.events[0].(domain.LineFailed)
	if !ok || failure.UtteranceID != "opening-2" || failure.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("failure = %#v", in.events[0])
	}
}

func TestOpeningExecutor_PreservesCallErrorAndAlias(t *testing.T) {
	llm := &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Err: &ports.CallError{Kind: vocab.ErrRateLimited}}}}
	in := &inbox{}
	NewOpeningExecutor(llm).StartLine(context.Background(), domain.StartLine{
		Role: vocab.RoleOpening, UtteranceID: "opening-rate", Input: "one-shot",
	}, domain.Scope{}, in)
	failure, ok := in.events[0].(domain.LineFailed)
	if !ok || failure.FailureKind != vocab.ErrRateLimited {
		t.Fatalf("failure = %#v", in.events[0])
	}
}

func TestOpeningExecutor_RejectsOverlongOutputAndNilLLM(t *testing.T) {
	long := "one two three four five six seven eight nine ten eleven twelve thirteen fourteen fifteen sixteen seventeen eighteen nineteen twenty twenty one twenty two twenty three twenty four twenty five twenty six twenty seven twenty eight twenty nine thirty thirty one thirty two thirty three thirty four thirty five thirty six thirty seven thirty eight thirty nine forty one"
	cases := []struct {
		name string
		llm  ports.LLM
	}{
		{name: "overlong", llm: &fakes.FakeLLM{Text: []fakes.LLMTextResult{{Chunks: []string{long}}}}},
		{name: "nil", llm: nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := &inbox{}
			NewOpeningExecutor(tc.llm).Execute(context.Background(), domain.StartLine{Role: vocab.RoleOpening, UtteranceID: "bad", Input: "one-shot"}, domain.Scope{}, in)
			if len(in.events) == 0 {
				t.Fatal("no result event")
			}
			if _, ok := in.events[len(in.events)-1].(domain.LineFailed); !ok {
				t.Fatalf("last event = %T, want LineFailed", in.events[len(in.events)-1])
			}
		})
	}
}

type nilStreamLLM struct{}

func (nilStreamLLM) StreamText(context.Context, ports.TextRequest) (ports.TextStream, error) {
	return nil, nil
}

func (nilStreamLLM) JSON(context.Context, ports.TextRequest, ports.Schema) (json.RawMessage, error) {
	return nil, nil
}

func TestOpeningExecutor_RejectsNilStream(t *testing.T) {
	in := &inbox{}
	NewOpeningExecutor(nilStreamLLM{}).Execute(context.Background(), domain.StartLine{Role: vocab.RoleOpening, UtteranceID: "nil-stream", Input: "one-shot"}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %d, want 1", len(in.events))
	}
	if failure := in.events[0].(domain.LineFailed); failure.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("failure = %#v", failure)
	}
}

func TestOpeningExecutor_RejectsWrongRoleAndBadInput(t *testing.T) {
	cases := []domain.StartLine{
		{Role: vocab.RoleNPCReply, UtteranceID: "wrong", Input: "x"},
		{Role: vocab.RoleOpening, UtteranceID: "empty"},
	}
	for _, effect := range cases {
		t.Run(string(effect.Role)+string(effect.UtteranceID), func(t *testing.T) {
			in := &inbox{}
			NewOpeningExecutor(&fakes.FakeLLM{}).Execute(context.Background(), effect, domain.Scope{}, in)
			if len(in.events) != 1 {
				t.Fatalf("events = %d, want 1", len(in.events))
			}
			if _, ok := in.events[0].(domain.LineFailed); !ok {
				t.Fatalf("event = %T, want LineFailed", in.events[0])
			}
		})
	}
}

func TestCharacterFlavorExecutor_PostsFlavor(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: []byte(`{"name":"Mira","look":"red cloak","hook":"find my brother","pronouns":"she/her"}`)}}}
	in := &inbox{}
	NewCharacterFlavorExecutor(llm).Execute(context.Background(), domain.CharacterFlavor{
		Seat: 2, Species: "human", Gender: "woman", Class: "paladin", Background: "sailor",
	}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %d, want 1", len(in.events))
	}
	done, ok := in.events[0].(domain.FlavorDone)
	if !ok || done.Seat != 2 || done.Flavor.Name != "Mira" || done.Flavor.Hook != "find my brother" {
		t.Fatalf("event = %#v", in.events[0])
	}
	if len(llm.JSONCalls) != 1 || llm.JSONCalls[0].Schema.Name != string(vocab.RoleCharacterFlavor) {
		t.Fatal("flavor schema was not sent")
	}
}

func TestCharacterFlavorExecutor_InvalidJSONPostsFailure(t *testing.T) {
	llm := &fakes.FakeLLM{JSONScript: []fakes.LLMJSONResult{{Value: []byte(`{"name":"Mira"}`)}}}
	in := &inbox{}
	NewCharacterFlavorExecutor(llm).Execute(context.Background(), domain.CharacterFlavor{Seat: 1, Species: "elf", Gender: "man", Class: "rogue", Background: "urchin"}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %d, want 1", len(in.events))
	}
	if failure, ok := in.events[0].(domain.FlavorFailed); !ok || failure.Seat != 1 {
		t.Fatalf("event = %#v", in.events[0])
	}
}

func TestCharacterFlavorExecutor_NilLLMPostsFailure(t *testing.T) {
	in := &inbox{}
	NewCharacterFlavorExecutor(nil).Execute(context.Background(), domain.CharacterFlavor{Seat: 1}, domain.Scope{}, in)
	if len(in.events) != 1 {
		t.Fatalf("events = %d, want 1", len(in.events))
	}
	if _, ok := in.events[0].(domain.FlavorFailed); !ok {
		t.Fatalf("event = %T, want FlavorFailed", in.events[0])
	}
}
