package elevenlabs

import (
	"context"
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

func TestAdapter_TranscribeMultipartAndFixture(t *testing.T) {
	fixture := readFixture(t, "response.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "secret" {
			t.Errorf("key=%q", r.Header.Get("xi-api-key"))
		}
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data;") {
			t.Errorf("content type=%q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("model_id") != scribeModel || !strings.EqualFold(r.FormValue("keyterms"), "Drowned Lantern") {
			t.Errorf("form=%v", r.MultipartForm.Value)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "webm-audio" || header.Filename != "audio.webm" {
			t.Errorf("file=%q name=%q", data, header.Filename)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()
	adapter := New("secret", server.URL, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	got, err := adapter.Transcribe(context.Background(), ports.STTRequest{Audio: []byte("webm-audio"), MIME: "audio/webm", Keyterms: []string{"Drowned Lantern"}})
	if err != nil || got.Text != "The lantern waits beneath the drowned tower." {
		t.Fatalf("transcript=%+v err=%v", got, err)
	}
}

func TestBuildRequest_RepeatsKeytermsAndSkipsBlank(t *testing.T) {
	body, contentType, err := buildRequest(ports.STTRequest{Audio: []byte("x"), MIME: "audio/ogg; codecs=opus", Keyterms: []string{"one", " ", "two"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(contentType, "multipart/form-data") || !strings.Contains(body.String(), `name="keyterms"`) {
		t.Fatalf("body=%q contentType=%q", body.String(), contentType)
	}
	if strings.Count(body.String(), `name="keyterms"`) != 2 || !strings.Contains(body.String(), "audio.ogg") {
		t.Fatalf("body=%q", body.String())
	}
}

func TestAdapter_MapsVendorFailuresAndBadResponses(t *testing.T) {
	tests := []struct {
		name string
		code int
		kind vocab.ErrKind
	}{
		{"auth", http.StatusUnauthorized, vocab.ErrAuth},
		{"rate", http.StatusTooManyRequests, vocab.ErrRateLimited},
		{"unprocessable", http.StatusUnprocessableEntity, vocab.ErrBadOutput},
		{"server", http.StatusBadGateway, vocab.ErrUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failure", tc.code) }))
			defer server.Close()
			_, err := New("key", server.URL, time.Second, nil).Transcribe(context.Background(), ports.STTRequest{Audio: []byte("x")})
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != tc.kind {
				t.Fatalf("err=%v", err)
			}
		})
	}
	for _, response := range []string{`{}`, `{"text":" "}`, `not-json`} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(response)) }))
		_, err := New("key", server.URL, time.Second, nil).Transcribe(context.Background(), ports.STTRequest{Audio: []byte("x")})
		server.Close()
		var callErr *ports.CallError
		if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
			t.Fatalf("response=%q err=%v", response, err)
		}
	}
}

func TestAdapter_RejectsEmptyAudio(t *testing.T) {
	_, err := New("key", "http://unused", time.Second, nil).Transcribe(context.Background(), ports.STTRequest{})
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
		t.Fatalf("err=%v", err)
	}
}

func TestParseResponseBytes(t *testing.T) {
	got, err := parseResponseBytes([]byte(`{"text":"hello"}`))
	if err != nil || got.Text != "hello" {
		t.Fatalf("got=%+v err=%v", got, err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return data
}
