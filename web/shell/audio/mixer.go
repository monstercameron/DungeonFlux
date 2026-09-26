package audio

import (
	"errors"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// Channel identifies a browser mixer bus.
type Channel string

const (
	// VoiceChannel carries streamed narration and dialogue.
	VoiceChannel Channel = "voice"
	// MusicChannel carries looping score.
	MusicChannel Channel = "music"
	// AmbienceChannel carries environmental beds.
	AmbienceChannel Channel = "ambience"
	// SFXChannel carries short sound effects.
	SFXChannel Channel = "sfx"
)

// TrackState is the platform-neutral state of one music, ambience, or SFX track.
type TrackState struct {
	ID      string
	Gain    float32
	Duck    float32
	Loop    bool
	Playing bool
}

// MixState is the deterministic command state used by the browser mixer.
// It contains no clock or browser objects and is safe to exercise in native tests.
type MixState struct {
	tracks map[string]TrackState
	voice  bool
}

// NewMixState creates an empty mixer state.
func NewMixState() *MixState { return &MixState{tracks: make(map[string]TrackState)} }

// Track returns a copy of a track's state and whether it exists.
func (m *MixState) Track(id string) (TrackState, bool) {
	if m == nil {
		return TrackState{}, false
	}
	track, ok := m.tracks[id]
	return track, ok
}

// VoiceActive reports whether a non-final voice stream is active.
func (m *MixState) VoiceActive() bool { return m != nil && m.voice }

// Apply updates mixer state from one protocol message.
func (m *MixState) Apply(message *dungeonfluxv1.AudioMessage) error {
	if m == nil || message == nil {
		return errors.New("audio: mixer message is required")
	}
	if m.tracks == nil {
		m.tracks = make(map[string]TrackState)
	}
	if frame := message.GetFrame(); frame != nil {
		if message.GetChannel() == dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_VOICE {
			m.voice = !frame.GetFinal()
		}
		return nil
	}
	if cancel := message.GetCancel(); cancel != nil {
		if cancel.GetAll() || cancel.GetUtteranceId() == "" {
			m.voice = false
		}
		return nil
	}
	if mix := message.GetMix(); mix != nil {
		if mix.GetTrackId() == "" {
			return errors.New("audio: mix command has no track id")
		}
		track := m.tracks[mix.GetTrackId()]
		track.ID, track.Gain, track.Duck = mix.GetTrackId(), mix.GetGain(), mix.GetDuck()
		track.Loop = mix.GetLoop()
		switch mix.GetKind() {
		case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY,
			dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_CROSSFADE:
			track.Playing = true
		case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_STOP:
			track.Playing = false
		case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_DUCK:
			track.Duck = mix.GetDuck()
		default:
			return errors.New("audio: unsupported mix command")
		}
		m.tracks[track.ID] = track
	}
	return nil
}

// DuckGain is the approximately 8 dB reduction used under voice.
const DuckGain float32 = 0.398
