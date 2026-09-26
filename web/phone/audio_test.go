package phone

import (
	"testing"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func sfxMessage(kind df.AudioTargetKind, seat int32, mime string) *df.AudioMessage {
	return &df.AudioMessage{
		Channel: df.AudioChannel_AUDIO_CHANNEL_SFX,
		Target:  &df.AudioTarget{Kind: kind, Seat: seat},
		Message: &df.AudioMessage_Chunk{Chunk: &df.EncodedAudioChunk{CodecMime: mime, Data: []byte{1}}},
	}
}

func TestAudioQueue_FiltersTargetAndPreservesOrder(t *testing.T) {
	q := NewAudioQueue()
	if q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT, 2, string(CueDamage)), 1) {
		t.Fatal("wrong seat was queued")
	}
	if !q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT, 1, string(CueRoll)), 1) {
		t.Fatal("seat cue was rejected")
	}
	if !q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES, 0, string(CueTurn)), 1) {
		t.Fatal("broadcast cue was rejected")
	}
	first, ok := q.Pop()
	if !ok || first.Cue != CueRoll {
		t.Fatalf("first cue = %#v, %v", first, ok)
	}
	second, ok := q.Pop()
	if !ok || second.Cue != CueTurn {
		t.Fatalf("second cue = %#v, %v", second, ok)
	}
}

func TestAudioQueue_MuteAndCapacityNeverBlock(t *testing.T) {
	q := NewAudioQueue()
	q.SetMuted(true)
	if q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES, 0, string(CueRoll)), 1) {
		t.Fatal("muted queue accepted cue")
	}
	q.SetMuted(false)
	for i := 0; i < audioQueueCapacity; i++ {
		if !q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES, 0, string(CueRoll)), 1) {
			t.Fatalf("cue %d was rejected before capacity", i)
		}
	}
	if q.Push(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES, 0, string(CueRoll)), 1) {
		t.Fatal("over-capacity cue was accepted")
	}
}

func TestPhoneAudio_ReducedMotionOnlyDisablesHaptics(t *testing.T) {
	audio := NewPhoneAudio()
	if !audio.HapticsEnabled() {
		t.Fatal("haptics should start enabled")
	}
	audio.SetReducedMotion(true)
	if audio.HapticsEnabled() {
		t.Fatal("reduced motion left haptics enabled")
	}
	if !audio.Enqueue(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES, 0, string(CueSuccess)), 1) {
		t.Fatal("reduced motion should not mute audio")
	}
}

func TestValidateAudioMessage_RejectsNonPhoneMessages(t *testing.T) {
	if validateAudioMessage(&df.AudioMessage{Channel: df.AudioChannel_AUDIO_CHANNEL_MUSIC}) == nil {
		t.Fatal("music message was accepted")
	}
	if validateAudioMessage(sfxMessage(df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT, 1, string(CueFailure))) != nil {
		t.Fatal("seat sfx was rejected")
	}
}
