package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type rehearsalCall struct {
	req    ports.TextRequest
	schema ports.Schema
	stream bool
}

func TestRecordReplay_IsolatesResponseContracts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*rehearsalCall)
	}{
		{"role", func(c *rehearsalCall) { c.req.Meta.Role = vocab.RoleInterpret }},
		{"language", func(c *rehearsalCall) { c.req.Meta.Locale = "es" }},
		{"schema name", func(c *rehearsalCall) { c.schema.Name = "other" }},
		{"schema shape", func(c *rehearsalCall) {
			c.schema.JSON = json.RawMessage(`{"type":"object","additionalProperties":false}`)
		}},
		{"response format", func(c *rehearsalCall) { c.stream = true; c.schema = ports.Schema{} }},
		{"phase", func(c *rehearsalCall) { c.req.Meta.Phase = vocab.StateResolution }},
		{"seat", func(c *rehearsalCall) { c.req.Meta.Seat = 2 }},
		{"call index", func(c *rehearsalCall) { c.req.Meta.Index = 2 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := rehearsalCall{req: ports.TextRequest{Meta: ports.CallMeta{Role: vocab.RoleNPCReply, Locale: "en", Phase: vocab.StateConversation, Seat: 1, Index: 1}}, schema: ports.Schema{Name: "reply", JSON: json.RawMessage(`{"type":"object"}`)}}
			second := first
			tc.change(&second)
			store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
			value, calls := `{"line":"first"}`, 0
			live := fakeLLM{
				jsonFn:   func(context.Context) (json.RawMessage, error) { calls++; return json.RawMessage(value), nil },
				streamFn: func(context.Context) (ports.TextStream, error) { calls++; return newFakeStream(value), nil },
			}
			decorator := RecordReplay(live, store, "sequence")
			runRehearsalCall(t, decorator, first, value)
			value = `{"line":"second"}`
			runRehearsalCall(t, decorator, second, value)
			first.req.Meta.ForceReplay, second.req.Meta.ForceReplay = true, true
			runRehearsalCall(t, decorator, first, `{"line":"first"}`)
			runRehearsalCall(t, decorator, second, `{"line":"second"}`)
			if calls != 2 || len(store.values) != 2 {
				t.Fatalf("live calls=%d stored entries=%d, want 2 each", calls, len(store.values))
			}
		})
	}
}

func runRehearsalCall(t *testing.T, llm ports.LLM, call rehearsalCall, want string) {
	t.Helper()
	if !call.stream {
		value, err := llm.JSON(context.Background(), call.req, call.schema)
		if err != nil || string(value) != want {
			t.Fatalf("JSON=%q error=%v, want %q", value, err, want)
		}
		return
	}
	stream, err := llm.StreamText(context.Background(), call.req)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var value strings.Builder
	for {
		text, err := stream.Recv()
		value.WriteString(text)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if value.String() != want {
		t.Fatalf("text=%q, want %q", value.String(), want)
	}
}

func TestRecordReplay_SequenceIgnoresPromptAndRunIdentity(t *testing.T) {
	store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
	calls := 0
	decorator := RecordReplay(fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"line":"saved"}`), nil
	}}, store, "sequence")
	call := rehearsalCall{req: ports.TextRequest{Meta: ports.CallMeta{Run: "rehearsal", Locale: "en"}, Messages: []ports.Message{{Text: "first wording"}}}}
	runRehearsalCall(t, decorator, call, `{"line":"saved"}`)
	call.req.Meta.Run, call.req.Meta.ForceReplay = "show", true
	call.req.Messages[0].Text = "different live wording"
	runRehearsalCall(t, decorator, call, `{"line":"saved"}`)
	if calls != 1 {
		t.Fatalf("forced sequence made %d live calls", calls)
	}
}

func TestRecordReplay_InvalidSchemaDoesNotCallProvider(t *testing.T) {
	called := false
	decorator := RecordReplay(fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		called = true
		return nil, nil
	}}, &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}, "sequence")
	_, err := decorator.JSON(context.Background(), ports.TextRequest{}, ports.Schema{JSON: json.RawMessage(`{`)})
	if err == nil || called {
		t.Fatalf("malformed schema: err=%v provider called=%v", err, called)
	}
}

func TestRecordReplay_DoesNotTrustAmbiguousLegacyEntry(t *testing.T) {
	store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{
		{Adapter: "sequence"}: {Text: `{"line":"possibly overwritten"}`},
	}}
	called := false
	decorator := RecordReplay(fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		called = true
		return nil, nil
	}}, store, "sequence")
	_, err := decorator.JSON(context.Background(), ports.TextRequest{Meta: ports.CallMeta{ForceReplay: true}}, ports.Schema{})
	if err == nil || called {
		t.Fatalf("legacy forced replay: err=%v provider called=%v", err, called)
	}
}
