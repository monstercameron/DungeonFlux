package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestAdapter_GenerateSendsTransparentRequestAndParsesFixture(t *testing.T) {
	fixture := readFixture(t, "response.json")
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	adapter := New("secret", server.URL, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	stream, err := adapter.Generate(context.Background(), ports.ImageRequest{Prompt: "a hero", Transparent: true})
	if err != nil {
		t.Fatal(err)
	}
	if received["background"] != "transparent" || received["model"] != model || received["output_format"] != "png" {
		t.Fatalf("request=%v", received)
	}
	event, err := stream.Recv()
	if err != nil || len(event.PNG) != len(pngFixture) || event.Partial {
		t.Fatalf("event=%+v err=%v", event, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second recv err=%v", err)
	}
}

func TestAdapter_GenerateParsesPartialFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"b64_json\":\"" + base64.StdEncoding.EncodeToString(pngFixture) + "\"}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()
	stream, err := New("key", server.URL, time.Second, nil).Generate(context.Background(), ports.ImageRequest{Prompt: "hero", Partials: 1})
	if err != nil {
		t.Fatal(err)
	}
	event, err := stream.Recv()
	if err != nil || !event.Partial || event.Index != 0 {
		t.Fatalf("event=%+v err=%v", event, err)
	}
}

func TestAdapter_GenerateMapsVendorFailures(t *testing.T) {
	tests := []struct {
		name string
		code int
		kind vocab.ErrKind
	}{
		{"auth", http.StatusUnauthorized, vocab.ErrAuth},
		{"rate", http.StatusTooManyRequests, vocab.ErrRateLimited},
		{"server", http.StatusBadGateway, vocab.ErrUnavailable},
		{"bad request", http.StatusBadRequest, vocab.ErrBadOutput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failure", tc.code) }))
			defer server.Close()
			_, err := New("key", server.URL, time.Second, nil).Generate(context.Background(), ports.ImageRequest{Prompt: "hero"})
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != tc.kind {
				t.Fatalf("err=%v", err)
			}
		})
	}
}

func TestAdapter_GenerateRejectsInvalidResponsesAndPrompt(t *testing.T) {
	_, err := New("key", "http://unused", time.Second, nil).Generate(context.Background(), ports.ImageRequest{})
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
		t.Fatalf("err=%v", err)
	}
	for _, body := range []string{`{"data":[]}`, `{"data":[{"b64_json":"not-base64"}]}`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(body)) }))
		_, err := New("key", server.URL, time.Second, nil).Generate(context.Background(), ports.ImageRequest{Prompt: "hero"})
		server.Close()
		if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
			t.Fatalf("body=%s err=%v", body, err)
		}
	}
}

func TestStream_CloseStopsReads(t *testing.T) {
	s := &stream{events: []ports.ImageEvent{{PNG: pngFixture}}}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("err=%v", err)
	}
}

var pngFixture = []byte{137, 80, 78, 71, 13, 10, 26, 10, 1}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	path := filepath.Join("testdata", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "PLACEHOLDER") {
		t.Fatal("invalid fixture")
	}
	return data
}
