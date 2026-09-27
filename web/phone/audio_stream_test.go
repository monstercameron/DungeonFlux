package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func chunkFor(seat int32, data byte, final bool) *df.AudioMessage {
	return &df.AudioMessage{Channel: df.AudioChannel_AUDIO_CHANNEL_SFX, Target: &df.AudioTarget{Kind: df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT, Seat: seat},
		Message: &df.AudioMessage_Chunk{Chunk: &df.EncodedAudioChunk{Data: []byte{data}, Final: final}}}
}

func playFor(seat int32, id string, delay int64, gain float32) *df.AudioMessage {
	return &df.AudioMessage{Channel: df.AudioChannel_AUDIO_CHANNEL_SFX, Target: &df.AudioTarget{Kind: df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT, Seat: seat},
		Message: &df.AudioMessage_Mix{Mix: &df.AudioMixCommand{Kind: df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY, TrackId: id, StartAtMs: delay, Gain: gain}}}
}

func TestSFXAssembler_joinsChunksWithTheirDelayedPlayCommand(t *testing.T) {
	var a sfxAssembler
	for _, message := range []*df.AudioMessage{chunkFor(1, 1, false), chunkFor(2, 9, true), chunkFor(1, 2, true)} {
		if _, ok := a.Accept(message, 1); ok {
			t.Fatal("a chunk alone must not play")
		}
	}
	if _, ok := a.Accept(playFor(2, "other", 0, 1), 1); ok {
		t.Fatal("another seat's command played")
	}
	got, ok := a.Accept(playFor(1, "sfx_phone_hurt", 2900, 0.9), 1)
	if !ok || got.TrackID != "sfx_phone_hurt" || got.DelayMS != 2900 || got.Gain != 0.9 || string(got.Data) != string([]byte{1, 2}) {
		t.Fatalf("effect = %#v, %v", got, ok)
	}
	if _, ok := a.Accept(playFor(1, "again", 0, 1), 1); ok {
		t.Fatal("a command without new chunks played")
	}
}

func TestSFXAssembler_newEffectStartsFreshAndDefaultsGain(t *testing.T) {
	var a sfxAssembler
	a.Accept(chunkFor(1, 7, true), 1)
	a.Accept(chunkFor(1, 8, true), 1)
	got, ok := a.Accept(playFor(1, "sfx_your_turn", -5, 0), 1)
	if !ok || string(got.Data) != string([]byte{8}) || got.DelayMS != 0 || got.Gain != 1 {
		t.Fatalf("effect = %#v, %v", got, ok)
	}
	music := &df.AudioMessage{Channel: df.AudioChannel_AUDIO_CHANNEL_MUSIC, Target: &df.AudioTarget{Kind: df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES}}
	if _, ok := a.Accept(music, 1); ok || phoneTarget(nil, 1) || phoneTarget(&df.AudioTarget{Kind: df.AudioTargetKind_AUDIO_TARGET_KIND_DM}, 1) {
		t.Fatal("non-phone messages must be ignored")
	}
	var nilAssembler *sfxAssembler
	if _, ok := nilAssembler.Accept(chunkFor(1, 1, true), 1); ok {
		t.Fatal("nil assembler accepted")
	}
}
