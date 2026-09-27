//go:build live

package openai

import (
	"context"
	"io"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type liveText struct{ done bool }

func (s *liveText) Recv() (string, error) {
	if s.done {
		return "", io.EOF
	}
	s.done = true
	return "Hi", nil
}
func (s *liveText) Close() error { return nil }

func TestLiveOpenAITTS(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_OPENAI_API_KEY")
	if key == "" {
		t.Fatal("DF_OPENAI_API_KEY is not set while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	stream, err := New(key, "", 45*time.Second, nil).Stream(ctx, ports.TTSRequest{VoiceID: "alloy"}, &liveText{})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk.S16LE) == 0 || chunk.SampleRate != 24000 {
		t.Fatalf("invalid OpenAI PCM: rate=%d bytes=%d", chunk.SampleRate, len(chunk.S16LE))
	}
}
