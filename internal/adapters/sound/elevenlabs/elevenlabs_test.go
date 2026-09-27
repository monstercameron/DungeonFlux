package elevenlabs

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestBuildRequest_SFXAndMusic(t *testing.T) {
	tests := []struct {
		name string
		kind vocab.SoundKind
		path string
	}{
		{name: "sfx", kind: vocab.SoundSFX, path: "/sound-generation"},
		{name: "music", kind: vocab.SoundMusic, path: "/music/detailed?output_format=mp3_44100_192"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path, body, err := buildRequest(ports.SoundRequest{Kind: tc.kind, Prompt: "rain", Seconds: 4})
			if err != nil || path != tc.path {
				t.Fatalf("path=%q err=%v", path, err)
			}
			var decoded map[string]any
			if err := json.Unmarshal(body, &decoded); err != nil {
				t.Fatal(err)
			}
			if tc.kind == vocab.SoundMusic && decoded["model_id"] != musicModel {
				t.Fatalf("model=%v", decoded["model_id"])
			}
		})
	}
}

func TestBuildRequest_RejectsInvalidInput(t *testing.T) {
	for name, request := range map[string]ports.SoundRequest{
		"missing prompt":   {Kind: vocab.SoundSFX, Seconds: 1},
		"missing duration": {Kind: vocab.SoundSFX, Prompt: "rain"},
		"unknown kind":     {Kind: "other", Prompt: "rain", Seconds: 1},
		"long music":       {Kind: vocab.SoundMusic, Prompt: "rain", Seconds: 601},
	} {
		t.Run(name, func(t *testing.T) {
			if _, _, err := buildRequest(request); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestAdapter_GeneratePostsExpectedRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/sound-generation" || r.Header.Get("xi-api-key") != "secret" {
			t.Fatalf("request path=%q key=%q", r.URL.Path, r.Header.Get("xi-api-key"))
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write([]byte("mp3"))
	}))
	defer server.Close()
	adapter := New("secret", server.URL+"/v1", httpx.NewClient("elevenlabs", 5_000_000_000, nil), nil)
	sound, err := adapter.Generate(context.Background(), ports.SoundRequest{Kind: vocab.SoundSFX, Prompt: "dice", Seconds: 2})
	if err != nil || string(sound.Bytes) != "mp3" || sound.DurationMS != 2000 {
		t.Fatalf("sound=%+v err=%v", sound, err)
	}
}

func TestAdapter_GenerateMapsHTTPAndContextErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()
	adapter := New("", server.URL, nil, nil)
	_, err := adapter.Generate(context.Background(), ports.SoundRequest{Kind: vocab.SoundSFX, Prompt: "x", Seconds: 1})
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
		t.Fatalf("err=%v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = adapter.Generate(ctx, ports.SoundRequest{Kind: vocab.SoundSFX, Prompt: "x", Seconds: 1})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("cancel err=%v", err)
	}
}

func TestParseResponse_RejectsEmptyAndNonOK(t *testing.T) {
	for name, response := range map[string]*http.Response{
		"nil":   nil,
		"empty": {StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(""))},
		"error": {StatusCode: http.StatusInternalServerError, Status: "500 Internal Server Error", Body: io.NopCloser(strings.NewReader("no"))},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseResponse(response, vocab.SoundSFX); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}
