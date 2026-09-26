//go:build live

package elevenlabs

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

func TestLiveElevenLabsTTS(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_ELEVENLABS_API_KEY")
	voice := os.Getenv("DF_PROBE_VOICE_ID")
	if key == "" || voice == "" {
		t.Fatal("DF_ELEVENLABS_API_KEY and DF_PROBE_VOICE_ID are required while DF_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	stream, err := New(key, "", nil).Stream(ctx, ports.TTSRequest{VoiceID: voice}, &liveText{})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	chunk, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	if len(chunk.S16LE) == 0 || chunk.SampleRate != 24000 {
		t.Fatalf("invalid ElevenLabs PCM: rate=%d bytes=%d", chunk.SampleRate, len(chunk.S16LE))
	}
}
