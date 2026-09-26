package anthropic

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestAdapter_StreamTextParsesFixtureAndRequest(t *testing.T) {
	fixture := readFixture(t, "stream.sse")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("path=%q", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "secret" || r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("headers=%v", r.Header)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	adapter := New("secret", server.URL, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	stream, err := adapter.StreamText(context.Background(), ports.TextRequest{
		Messages: []ports.Message{{Role: vocab.MsgSystem, Text: "Speak briefly."}, {Role: vocab.MsgUser, Text: "Greet me."}}, MaxTokens: 12,
	})
	if err != nil {
		t.Fatal(err)
	}
	var got string
	for {
		part, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			t.Fatal(recvErr)
		}
		got += part
	}
	if got != "Hello there." {
		t.Fatalf("text=%q", got)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAdapter_StreamTextMapsProviderFailures(t *testing.T) {
	tests := []struct {
		name  string
		code  int
		kind  vocab.ErrKind
		retry bool
	}{
		{name: "auth", code: http.StatusUnauthorized, kind: vocab.ErrAuth},
		{name: "rate limited", code: http.StatusTooManyRequests, kind: vocab.ErrRateLimited, retry: true},
		{name: "server", code: http.StatusBadGateway, kind: vocab.ErrUnavailable, retry: true},
		{name: "bad request", code: http.StatusBadRequest, kind: vocab.ErrBadOutput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("request-id", "req_test")
				http.Error(w, `{"error":{"type":"api_error","message":"failure"}}`, tc.code)
			}))
			defer server.Close()
			_, err := New("key", server.URL, time.Second, nil).StreamText(context.Background(), userRequest())
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != tc.kind || callErr.Retryable != tc.retry || callErr.RequestID != "req_test" {
				t.Fatalf("err=%+v", err)
			}
		})
	}
}

func TestAdapter_RejectsInvalidRequestsAndJSON(t *testing.T) {
	adapter := New("key", "http://unused", time.Second, nil)
	for _, request := range []ports.TextRequest{{}, {Messages: []ports.Message{{Role: vocab.MsgSystem, Text: "only system"}}}, {Messages: []ports.Message{{Role: vocab.MsgRole("other"), Text: "bad"}}}} {
		_, err := adapter.StreamText(context.Background(), request)
		var callErr *ports.CallError
		if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
			t.Fatalf("request=%+v err=%v", request, err)
		}
	}
	_, err := adapter.JSON(context.Background(), userRequest(), ports.Schema{Name: "unused"})
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrUnavailable {
		t.Fatalf("json err=%v", err)
	}
}

func userRequest() ports.TextRequest {
	return ports.TextRequest{Messages: []ports.Message{{Role: vocab.MsgUser, Text: "hello"}}}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
