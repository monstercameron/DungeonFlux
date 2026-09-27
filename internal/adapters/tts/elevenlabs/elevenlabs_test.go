package elevenlabs

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type textStream struct {
	values []string
	index  int
}

func (s *textStream) Recv() (string, error) {
	if s.index == len(s.values) {
		return "", io.EOF
	}
	value := s.values[s.index]
	s.index++
	return value, nil
}

func (s *textStream) Close() error { return nil }

func TestAdapter_StreamFramesAndProtocol(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("xi-api-key") != "secret" {
			t.Errorf("API key header=%q", r.Header.Get("xi-api-key"))
		}
		if r.URL.Query().Get("model_id") != modelID || r.URL.Query().Get("output_format") != "pcm_24000" || r.URL.Query().Get("auto_mode") != "true" {
			t.Errorf("query=%v", r.URL.Query())
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		var messages []map[string]any
		for len(messages) < 3 {
			var message map[string]any
			if err := conn.ReadJSON(&message); err != nil {
				t.Error(err)
				return
			}
			messages = append(messages, message)
			if len(messages) == 3 {
				break
			}
		}
		if messages[0]["text"] != " " || messages[1]["text"] != "hello" || messages[2]["text"] != "" {
			t.Errorf("messages=%v", messages)
		}
		pcm := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4})
		_ = conn.WriteJSON(map[string]any{"audio": pcm})
		_ = conn.WriteJSON(map[string]any{"audio": "", "isFinal": true})
	}))
	defer server.Close()
	adapter := New("secret", "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	stream, err := adapter.Stream(t.Context(), ports.TTSRequest{VoiceID: "voice/one"}, &textStream{values: []string{"hello"}})
	if err != nil {
		t.Fatal(err)
	}
	frame, err := stream.Recv()
	if err != nil || frame.SampleRate != sampleRate || string(frame.S16LE) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("frame=%+v err=%v", frame, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("final err=%v", err)
	}
	_ = stream.Close()
}

func TestAdapter_StreamRejectsInvalidRequests(t *testing.T) {
	adapter := New("key", "", nil)
	for name, req := range map[string]ports.TTSRequest{"missing_voice": {}, "missing_text": {VoiceID: "v"}} {
		t.Run(name, func(t *testing.T) {
			var text ports.TextStream
			if name == "missing_text" {
				text = nil
			}
			if _, err := adapter.Stream(context.Background(), req, text); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestParseMessage(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []byte
		fail bool
	}{
		{name: "audio", body: `{"audio":"AQI=","isFinal":false}`, want: []byte{1, 2}},
		{name: "final", body: `{"audio":"","isFinal":true}`},
		{name: "bad_json", body: `{`, fail: true},
		{name: "bad_audio", body: `{"audio":"!"}`, fail: true},
		{name: "vendor_error", body: `{"error":"bad voice"}`, fail: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := parseMessage([]byte(tt.body))
			if (err != nil) != tt.fail || string(message.Audio) != string(tt.want) {
				t.Fatalf("message=%+v err=%v", message, err)
			}
		})
	}
}

func TestStreamURL(t *testing.T) {
	got, err := streamURL("ws://localhost/base", "voice-one")
	if err != nil || !strings.Contains(got, "/base/voice-one/stream-input") {
		t.Fatalf("url=%q err=%v", got, err)
	}
	if _, err := streamURL("http://localhost", "voice"); err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestAdapter_StreamContextCancellation(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err == nil {
			defer conn.Close()
			_, _, _ = conn.ReadMessage()
		}
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	adapter := New("key", "ws"+strings.TrimPrefix(server.URL, "http"), nil)
	stream, err := adapter.Stream(ctx, ports.TTSRequest{VoiceID: "voice"}, &textStream{values: []string{"x"}})
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	_ = stream.Close()
}
