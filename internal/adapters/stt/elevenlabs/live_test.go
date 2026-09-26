//go:build live

package elevenlabs

import (
	"bytes"
	"context"
	"encoding/binary"
	"os"
	"strings"
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
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	result, err := New(key, "", 45*time.Second, nil).Transcribe(ctx, ports.STTRequest{Audio: silenceWAV(), MIME: "audio/wav"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.IndexByte(result.Text, 0) >= 0 {
		t.Fatal("Scribe transcript contains a NUL byte")
	}
}

func silenceWAV() []byte {
	const samples = 16000
	b := bytes.NewBuffer(nil)
	b.WriteString("RIFF")
	binary.Write(b, binary.LittleEndian, uint32(36+samples*2))
	b.WriteString("WAVEfmt ")
	binary.Write(b, binary.LittleEndian, uint32(16))
	binary.Write(b, binary.LittleEndian, uint16(1))
	binary.Write(b, binary.LittleEndian, uint16(1))
	binary.Write(b, binary.LittleEndian, uint32(16000))
	binary.Write(b, binary.LittleEndian, uint32(32000))
	binary.Write(b, binary.LittleEndian, uint16(2))
	binary.Write(b, binary.LittleEndian, uint16(16))
	b.WriteString("data")
	binary.Write(b, binary.LittleEndian, uint32(samples*2))
	b.Write(make([]byte, samples*2))
	return b.Bytes()
}
