package audio

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestMixState_CommandsTrackPlaybackAndCrossfade(t *testing.T) {
	m := NewMixState()
	for _, kind := range []dungeonfluxv1.AudioMixCommandKind{
		dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY,
		dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_CROSSFADE,
	} {
		if err := m.Apply(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Mix{Mix: &dungeonfluxv1.AudioMixCommand{Kind: kind, TrackId: "tavern", Gain: .6, Loop: true}}}); err != nil {
			t.Fatal(err)
		}
	}
	track, ok := m.Track("tavern")
	if !ok || !track.Playing || !track.Loop || track.Gain != .6 {
		t.Fatalf("track = %#v, ok=%v", track, ok)
	}
	if err := m.Apply(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Mix{Mix: &dungeonfluxv1.AudioMixCommand{Kind: dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_STOP, TrackId: "tavern"}}}); err != nil {
		t.Fatal(err)
	}
	track, _ = m.Track("tavern")
	if track.Playing {
		t.Fatal("stop left track playing")
	}
}

func TestMixState_VoiceFramesAndCancelUpdateVoiceState(t *testing.T) {
	m := NewMixState()
	frame := func(final bool) *dungeonfluxv1.AudioMessage {
		return &dungeonfluxv1.AudioMessage{Channel: dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_VOICE, Message: &dungeonfluxv1.AudioMessage_Frame{Frame: &dungeonfluxv1.AudioFrame{UtteranceId: "line", SampleRate: 24000, Final: final}}}
	}
	if err := m.Apply(frame(false)); err != nil || !m.VoiceActive() {
		t.Fatalf("voice start: err=%v active=%v", err, m.VoiceActive())
	}
	if err := m.Apply(frame(true)); err != nil || m.VoiceActive() {
		t.Fatalf("voice finish: err=%v active=%v", err, m.VoiceActive())
	}
	if err := m.Apply(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Cancel{Cancel: &dungeonfluxv1.AudioCancel{All: true}}}); err != nil {
		t.Fatal(err)
	}
}

func TestMixState_RejectsInvalidMix(t *testing.T) {
	if err := NewMixState().Apply(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Mix{Mix: &dungeonfluxv1.AudioMixCommand{}}}); err == nil {
		t.Fatal("missing track accepted")
	}
	if err := NewMixState().Apply(nil); err == nil {
		t.Fatal("nil message accepted")
	}
}

func TestMixState_DuckCommandPreservesPlayback(t *testing.T) {
	m := NewMixState()
	play := &dungeonfluxv1.AudioMixCommand{Kind: dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY, TrackId: "music", Gain: .6}
	duck := &dungeonfluxv1.AudioMixCommand{Kind: dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_DUCK, TrackId: "music", Duck: DuckGain}
	for _, command := range []*dungeonfluxv1.AudioMixCommand{play, duck} {
		if err := m.Apply(&dungeonfluxv1.AudioMessage{Message: &dungeonfluxv1.AudioMessage_Mix{Mix: command}}); err != nil {
			t.Fatal(err)
		}
	}
	track, _ := m.Track("music")
	if !track.Playing || track.Duck != DuckGain {
		t.Fatalf("track = %#v", track)
	}
}
