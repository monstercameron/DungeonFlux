package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestAdapterJSON_BuildsLowThinkingSchemaAndParsesFixture(t *testing.T) {
	var request map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "secret" {
			t.Errorf("api key header=%q", r.Header.Get("x-goog-api-key"))
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, w, `{"candidates":[{"content":{"parts":[{"text":"{\"line\":\"By the river.\"}"}]}}]}`)
	}))
	defer server.Close()

	schema := ports.Schema{Name: "line", JSON: json.RawMessage(`{"type":"object","properties":{"line":{"type":"string"}},"required":["line"]}`)}
	result, err := New("secret", server.URL+"/", time.Second).JSON(context.Background(), ports.TextRequest{
		Messages: []ports.Message{{Role: vocab.MsgSystem, Text: "Be terse."}, {Role: vocab.MsgUser, Text: "Speak."}}, MaxTokens: 40,
	}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"line":"By the river."}` {
		t.Fatalf("result=%s", result)
	}
	config := request["generationConfig"].(map[string]any)
	if config["responseMimeType"] != "application/json" {
		t.Fatalf("config=%v", config)
	}
	thinking := config["thinkingConfig"].(map[string]any)
	if thinking["thinkingLevel"] != "LOW" {
		t.Fatalf("thinking=%v", thinking)
	}
	system := request["systemInstruction"].(map[string]any)
	if system["parts"].([]any)[0].(map[string]any)["text"] != "Be terse." {
		t.Fatalf("system instruction=%v", system)
	}
	if config["responseJsonSchema"].(map[string]any)["title"] != nil {
		t.Fatalf("schema unexpectedly altered: %v", config["responseJsonSchema"])
	}
}

func TestAdapterStreamText_ReturnsResponseAndEOF(t *testing.T) {
	server := fixtureServer(`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`)
	defer server.Close()
	stream, err := New("key", server.URL+"/", time.Second).StreamText(context.Background(), ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgSystem, Text: "ignored"}, {Role: vocab.MsgAssistant, Text: "prior"}, {Role: vocab.MsgUser, Text: "say"}}})
	if err != nil {
		t.Fatal(err)
	}
	value, err := stream.Recv()
	if err != nil || value != "hello" {
		t.Fatalf("value=%q err=%v", value, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second recv=%v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAdapterJSON_MapsHTTPFailures(t *testing.T) {
	tests := []struct {
		name string
		code int
		kind vocab.ErrKind
	}{
		{"auth", http.StatusUnauthorized, vocab.ErrAuth},
		{"rate", http.StatusTooManyRequests, vocab.ErrRateLimited},
		{"bad request", http.StatusBadRequest, vocab.ErrBadOutput},
		{"server", http.StatusBadGateway, vocab.ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failure", tc.code) }))
			defer server.Close()
			_, err := New("key", server.URL+"/", time.Second).JSON(context.Background(), ports.TextRequest{}, ports.Schema{JSON: json.RawMessage(`{"type":"object"}`)})
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != tc.kind || callErr.Vendor != vocab.VendorGemini {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestAdapterJSON_RejectsBadSchemaAndEmptyResponse(t *testing.T) {
	_, err := New("key", "http://unused/", time.Second).JSON(context.Background(), ports.TextRequest{}, ports.Schema{JSON: json.RawMessage(`{`)})
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
		t.Fatalf("schema err=%v", err)
	}
	server := fixtureServer(`{"candidates":[]}`)
	defer server.Close()
	_, err = New("key", server.URL+"/", time.Second).JSON(context.Background(), ports.TextRequest{}, ports.Schema{JSON: json.RawMessage(`{"type":"object"}`)})
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
		t.Fatalf("empty err=%v", err)
	}
}

func fixtureServer(body string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeFixture(nil, w, body) }))
}

func writeFixture(t *testing.T, w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.WriteString(w, body); err != nil && t != nil {
		t.Fatal(err)
	}
}
