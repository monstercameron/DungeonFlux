package schemaflux

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestAdapter_JSONStrictSchemaAndEffort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["reasoning"].(map[string]any)["effort"] != "low" {
			t.Errorf("reasoning = %#v", body["reasoning"])
		}
		format := body["text"].(map[string]any)["format"].(map[string]any)
		if format["type"] != "json_schema" || format["strict"] != true {
			t.Errorf("format = %#v", format)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"r1","status":"completed","model":"gpt-6-luna","output":[{"type":"message","content":[{"type":"output_text","text":"{\"ok\":true}"}]}]}`)
	}))
	defer server.Close()
	a, err := New(Config{Provider: "openai", APIKey: "key", BaseURL: server.URL, Model: "gpt-6-luna", ReasoningEffort: "low"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.JSON(context.Background(), ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgSystem, Text: "sys"}, {Role: vocab.MsgUser, Text: "hi"}}}, ports.Schema{Name: "answer", JSON: json.RawMessage(`{"type":"object","properties":{"ok":{"type":"boolean"}},"required":["ok"],"additionalProperties":false}`)})
	if err != nil || string(got) != `{"ok":true}` {
		t.Fatalf("JSON = %s, %v", got, err)
	}
}

func TestAdapter_StreamText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"hello \"}\n\n")
		io.WriteString(w, "data: {\"type\":\"response.output_text.delta\",\"delta\":\"world\"}\n\n")
		io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"hello world\"}]}]}}\n\n")
	}))
	defer server.Close()
	a, err := NewOpenAI("key", server.URL, "gpt-6-luna", "none", nil)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := a.StreamText(context.Background(), ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "hi"}}})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	var b strings.Builder
	for {
		part, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			t.Fatal(recvErr)
		}
		b.WriteString(part)
	}
	if b.String() != "hello world" {
		t.Fatalf("stream = %q", b.String())
	}
}

func TestAdapter_CerebrasStrictJSONAndNoStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["reasoning_effort"] != "none" {
			t.Errorf("effort = %#v", body["reasoning_effort"])
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"model":"qwen","choices":[{"finish_reason":"stop","message":{"content":"{\"ok\":true}"}}]}`)
	}))
	defer server.Close()
	a, err := New(Config{Provider: "cerebras", APIKey: "key", BaseURL: server.URL, Model: "qwen"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.JSON(context.Background(), ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "hi"}}}, ports.Schema{Name: "answer", JSON: json.RawMessage(`{"type":"object"}`)})
	if err != nil || string(got) != `{"ok":true}` {
		t.Fatalf("JSON = %s, %v", got, err)
	}
	if _, err := a.StreamText(context.Background(), ports.TextRequest{}); err == nil {
		t.Fatal("expected unsupported streaming error")
	}
}
