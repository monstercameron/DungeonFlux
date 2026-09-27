//go:build live

package elevenlabs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

func TestLiveElevenLabsSTT(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("live test requires DF_LIVE=1")
	}
	key := os.Getenv("DF_ELEVENLABS_API_KEY")
	if key == "" {
		t.Fatal("DF_ELEVENLABS_API_KEY is not set while DF_LIVE=1")
	}
	audio, err := liveSpeechAudio()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	result, err := New(key, "", 45*time.Second, nil).Transcribe(ctx, ports.STTRequest{Audio: audio, MIME: "audio/wav"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Text == "" {
		t.Fatal("Scribe returned an empty transcript")
	}
}

func liveSpeechAudio() ([]byte, error) {
	path := os.Getenv("DF_LIVE_STT_AUDIO_FILE")
	if path == "" {
		return nil, fmt.Errorf("DF_LIVE_STT_AUDIO_FILE is required while DF_LIVE=1")
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat live STT audio: %w", err)
	}
	const maxAudioBytes = 10 << 20
	if info.Size() < 44 || info.Size() > maxAudioBytes {
		return nil, fmt.Errorf("live STT audio size %d is outside 44..%d bytes", info.Size(), maxAudioBytes)
	}
	audio, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read live STT audio: %w", err)
	}
	if !bytes.Equal(audio[:4], []byte("RIFF")) || !bytes.Equal(audio[8:12], []byte("WAVE")) {
		return nil, fmt.Errorf("live STT audio is not a RIFF/WAVE file")
	}
	return audio, nil
}
