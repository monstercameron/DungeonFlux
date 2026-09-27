package media

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type audioSinkFake struct {
	chunks []AudioChunk
	mixes  []MixCommand
}

func (s *audioSinkFake) PublishChunk(chunk AudioChunk) { s.chunks = append(s.chunks, chunk) }
func (s *audioSinkFake) PublishMix(mix MixCommand)     { s.mixes = append(s.mixes, mix) }

func TestAudioRouter_RouteAssetChunksAndMarksFinal(t *testing.T) {
	sink := &audioSinkFake{}
	router, err := NewAudioRouter(sink, 3)
	if err != nil {
		t.Fatal(err)
	}
	err = router.RouteAsset(domain.Asset{MIME: "audio/mpeg", DurationMS: 900}, []byte("abcdefg"), vocab.SoundMusic, AudioTarget{Kind: "dm"})
	if err != nil {
		t.Fatal(err)
	}
	if len(sink.chunks) != 3 || string(sink.chunks[2].Data) != "g" || !sink.chunks[2].Final {
		t.Fatalf("chunks=%+v", sink.chunks)
	}
	if sink.chunks[0].CodecMIME != "audio/mpeg" || sink.chunks[0].DurationMS != 385 {
		t.Fatalf("metadata=%+v", sink.chunks[0])
	}
}

func TestAudioRouter_RouteSoundPublishesPlayMix(t *testing.T) {
	sink := &audioSinkFake{}
	router, err := NewAudioRouter(sink, 10)
	if err != nil {
		t.Fatal(err)
	}
	err = router.RouteSound(domain.PlaySound{Channel: vocab.SoundSFX, Name: "dice", Target: "seat", Seat: 2, Loop: false, Gain: .8}, domain.Asset{MIME: "audio/ogg;codecs=opus"}, []byte("dice"))
	if err != nil || len(sink.chunks) != 1 || len(sink.mixes) != 1 {
		t.Fatalf("err=%v chunks=%d mixes=%d", err, len(sink.chunks), len(sink.mixes))
	}
	if sink.mixes[0].Target.Seat != 2 || sink.mixes[0].Command != "play" || sink.mixes[0].Gain != .8 {
		t.Fatalf("mix=%+v", sink.mixes[0])
	}
}

func TestAudioRouter_RouteMusicAssetPublishesChunksBeforeCrossfade(t *testing.T) {
	sink := &audioSinkFake{}
	router, err := NewAudioRouter(sink, 10)
	if err != nil {
		t.Fatal(err)
	}
	err = router.RouteMusicAsset(domain.MusicTrack{ID: "combat", Asset: "combat-audio", Level: .5, Duck: .2}, []byte("track"), AudioTarget{Kind: "dm"}, MusicCue{Kind: CueLoopTransition, Track: "combat", StartAtMS: 3000, CrossfadeMS: 1500})
	if err != nil || len(sink.chunks) != 1 || len(sink.mixes) != 1 {
		t.Fatalf("err=%v chunks=%d mixes=%d", err, len(sink.chunks), len(sink.mixes))
	}
	if sink.mixes[0].Command != "crossfade" || sink.mixes[0].StartAtMS != 3000 || !sink.mixes[0].Loop {
		t.Fatalf("mix=%+v", sink.mixes[0])
	}
}

func TestAudioRouter_RejectsInvalidAndSkipsNoopTransition(t *testing.T) {
	if _, err := NewAudioRouter(nil, 1); err == nil {
		t.Fatal("nil sink accepted")
	}
	sink := &audioSinkFake{}
	router, err := NewAudioRouter(sink, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := router.RouteAsset(domain.Asset{}, nil, vocab.SoundAmbience, AudioTarget{Kind: "dm"}); err != nil {
		t.Fatal(err)
	}
	if sink.chunks[0].CodecMIME != "audio/ogg;codecs=opus" || !sink.chunks[0].Final {
		t.Fatalf("empty chunk=%+v", sink.chunks[0])
	}
	if err := router.RouteMusicTransition(MusicCue{Kind: CueNone}, AudioTarget{}, 1, 1); err != nil || len(sink.mixes) != 0 {
		t.Fatalf("noop transition err=%v mixes=%d", err, len(sink.mixes))
	}
}
