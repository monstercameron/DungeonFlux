package wire

import (
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSequenceTTS_RecordsRuntimePositionAndForcesOfflineReplay(t *testing.T) {
	store := positionedRecordingStore{}
	live := &fakes.FakeTTS{Script: []fakes.TTSResult{{Chunks: []ports.PCMChunk{{SampleRate: 24000, S16LE: []byte{1, 2}}}}}}
	ctx := ports.WithCallMeta(t.Context(), ports.CallMeta{Phase: vocab.StateConversation, Seat: 2, Index: 7, Role: vocab.RoleNPCReply})
	req := ports.TTSRequest{VoiceID: "vell", SampleRate: 24000, Meta: ports.CallMeta{Locale: "es"}}
	stream, err := sequenceTTS(config.Config{}, live, store).Stream(ctx, req, safeStream{})
	if err != nil {
		t.Fatal(err)
	}
	if chunk, err := stream.Recv(); err != nil || len(chunk.S16LE) != 2 {
		t.Fatalf("live audio=%+v %v", chunk, err)
	}
	if _, err := stream.Recv(); err != io.EOF {
		t.Fatal(err)
	}
	_ = stream.Close()
	meta := live.Calls[0].Request.Meta
	if meta.Phase != vocab.StateConversation || meta.Seat != 2 || meta.Index != 7 || meta.Locale != "es" {
		t.Fatalf("metadata=%+v", meta)
	}
	cfg := config.Config{}
	cfg.Features.SequenceMode = true
	spy := &fakes.FakeTTS{}
	stream, err = sequenceTTS(cfg, spy, store).Stream(ctx, req, safeStream{})
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if chunk, err := stream.Recv(); err != nil || len(chunk.S16LE) != 2 {
		t.Fatalf("replayed audio=%+v %v", chunk, err)
	}
	if len(spy.Calls) != 0 {
		t.Fatal("Safe Mode called TTS")
	}
	if _, err := sequenceTTS(cfg, spy, nil).Stream(ctx, req, safeStream{}); err == nil || len(spy.Calls) != 0 {
		t.Fatal("missing store called TTS")
	}
}
