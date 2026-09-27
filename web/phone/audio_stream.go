package phone

import df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// StreamedSFX is one server-pushed effect for this phone, assembled from its
// encoded chunks and the play command that follows them.
type StreamedSFX struct {
	TrackID string
	Data    []byte
	DelayMS int64
	Gain    float32
}

// sfxAssembler joins a seat's SFX chunks with the mixer command that names
// the effect, its gain, and its start delay. The engine delays a hurt thud
// to the TV's contact frame, so the phone must honour the delay too.
type sfxAssembler struct {
	pending []byte
	final   bool
}

// Accept feeds one Listen message and returns an effect when its play
// command arrives after a complete set of chunks. Messages for other seats,
// other channels, and stray commands are ignored.
func (a *sfxAssembler) Accept(message *df.AudioMessage, seat int32) (StreamedSFX, bool) {
	if a == nil || message == nil || message.GetChannel() != df.AudioChannel_AUDIO_CHANNEL_SFX || !phoneTarget(message.GetTarget(), seat) {
		return StreamedSFX{}, false
	}
	if chunk := message.GetChunk(); chunk != nil {
		if a.final {
			a.pending, a.final = nil, false
		}
		a.pending = append(a.pending, chunk.GetData()...)
		a.final = chunk.GetFinal()
		return StreamedSFX{}, false
	}
	mix := message.GetMix()
	if mix == nil || mix.GetKind() != df.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY || !a.final || len(a.pending) == 0 {
		return StreamedSFX{}, false
	}
	effect := StreamedSFX{TrackID: mix.GetTrackId(), Data: a.pending, DelayMS: max(mix.GetStartAtMs(), 0), Gain: mix.GetGain()}
	if effect.Gain <= 0 {
		effect.Gain = 1
	}
	a.pending, a.final = nil, false
	return effect, true
}

func phoneTarget(target *df.AudioTarget, seat int32) bool {
	if target == nil {
		return false
	}
	switch target.GetKind() {
	case df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES:
		return true
	case df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT:
		return seat > 0 && target.GetSeat() == seat
	default:
		return false
	}
}
