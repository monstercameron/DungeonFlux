package openai

import (
	"context"
	"errors"
	"io"
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

func TestAdapter_StreamBuildsRequestAndReturnsFixture(t *testing.T) {
	fixture := readFixture(t, "speech.pcm")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/audio/speech" {
			t.Fatalf("request=%s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{`"input":"hello world"`, `"model":"gpt-4o-mini-tts"`, `"voice":"alloy"`, `"response_format":"pcm"`} {
			if !strings.Contains(string(body), want) {
				t.Fatalf("body=%s missing %s", body, want)
			}
		}
		w.Header().Set("Content-Type", "audio/pcm")
		_, _ = w.Write(fixture)
	}))
	defer server.Close()

	stream, err := New("secret", server.URL, time.Second, nil).Stream(context.Background(), ports.TTSRequest{VoiceID: "alloy"}, &textStream{parts: []string{"hello ", "world"}})
	if err != nil {
		t.Fatal(err)
	}
	chunk, err := stream.Recv()
	if err != nil || chunk.SampleRate != outputRate || string(chunk.S16LE) != string(fixture) {
		t.Fatalf("chunk=%+v err=%v", chunk, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("second recv err=%v", err)
	}
}

func TestAdapter_StreamMapsInputAndTransportFailures(t *testing.T) {
	_, err := New("key", "http://unused", time.Second, nil).Stream(context.Background(), ports.TTSRequest{VoiceID: "alloy"}, nil)
	assertKind(t, err, vocab.ErrUnavailable)
	for _, tc := range []struct {
		name string
		code int
		kind vocab.ErrKind
	}{
		{name: "auth", code: http.StatusUnauthorized, kind: vocab.ErrUnavailable},
		{name: "bad request", code: http.StatusBadRequest, kind: vocab.ErrUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "failure", tc.code) }))
			defer server.Close()
			_, err := New("key", server.URL, time.Second, nil).Stream(context.Background(), ports.TTSRequest{VoiceID: "alloy"}, &textStream{parts: []string{"hello"}})
			assertKind(t, err, tc.kind)
		})
	}
}

func TestAdapter_StreamRejectsEmptyInputAndResponse(t *testing.T) {
	for _, tc := range []struct {
		name  string
		voice string
		text  *textStream
	}{
		{name: "empty text", voice: "alloy", text: &textStream{}},
		{name: "empty voice", voice: "", text: &textStream{parts: []string{"hello"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := New("key", "http://unused", time.Second, nil).Stream(context.Background(), ports.TTSRequest{VoiceID: tc.voice}, tc.text)
			assertKind(t, err, vocab.ErrBadOutput)
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()
	_, err := New("key", server.URL, time.Second, nil).Stream(context.Background(), ports.TTSRequest{VoiceID: "alloy"}, &textStream{parts: []string{"hello"}})
	assertKind(t, err, vocab.ErrBadOutput)
}

func TestStream_CloseStopsReads(t *testing.T) {
	s := &stream{chunk: ports.PCMChunk{SampleRate: outputRate, S16LE: []byte{1}}}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("err=%v", err)
	}
}

type textStream struct {
	parts []string
}

func (s *textStream) Recv() (string, error) {
	if len(s.parts) == 0 {
		return "", io.EOF
	}
	part := s.parts[0]
	s.parts = s.parts[1:]
	return part, nil
}

func (s *textStream) Close() error { return nil }

func assertKind(t *testing.T, err error, want vocab.ErrKind) {
	t.Helper()
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != want {
		t.Fatalf("err=%v", err)
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
