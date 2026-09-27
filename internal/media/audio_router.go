package media

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// AudioTarget identifies the audience for a cue without depending on the API
// transport package.
type AudioTarget struct {
	Kind string
	Seat domain.SeatID
}

// AudioChunk is an encoded segment ready for the Listen stream.
type AudioChunk struct {
	Channel    vocab.SoundKind
	Target     AudioTarget
	CodecMIME  string
	Seq        uint64
	Data       []byte
	Final      bool
	DurationMS int
}

// MixCommand is a client-side mixer instruction associated with a channel.
type MixCommand struct {
	Channel    vocab.SoundKind
	Target     AudioTarget
	Command    string
	TrackID    string
	StartAtMS  int64
	DurationMS int
	Loop       bool
	Gain       float32
	Duck       float32
}

// AudioSink receives chunks and mix commands. The wire package adapts these
// values to the API's bounded Listen hub.
type AudioSink interface {
	PublishChunk(AudioChunk)
	PublishMix(MixCommand)
}

// AudioRouter turns manifest bytes and engine sound effects into streamable
// chunks while keeping the transport out of the media package.
type AudioRouter struct {
	sink       AudioSink
	chunkBytes int
}

// NewAudioRouter constructs a router with a bounded encoded chunk size.
func NewAudioRouter(sink AudioSink, chunkBytes int) (*AudioRouter, error) {
	if sink == nil {
		return nil, errors.New("audio router: sink is required")
	}
	if chunkBytes <= 0 {
		return nil, errors.New("audio router: chunk size must be positive")
	}
	return &AudioRouter{sink: sink, chunkBytes: chunkBytes}, nil
}

// RouteAsset publishes an asset's bytes as ordered encoded chunks. The final
// chunk is marked even when the asset is empty, so clients can complete a
// segment without relying on a connection close.
func (r *AudioRouter) RouteAsset(asset domain.Asset, data []byte, channel vocab.SoundKind, target AudioTarget) error {
	if r == nil || r.sink == nil {
		return errors.New("audio router: router is not configured")
	}
	if len(data) == 0 {
		r.sink.PublishChunk(AudioChunk{Channel: channel, Target: target, CodecMIME: codecMIME(asset.MIME), Final: true, DurationMS: asset.DurationMS})
		return nil
	}
	for seq, offset := uint64(0), 0; offset < len(data); seq++ {
		end := min(offset+r.chunkBytes, len(data))
		r.sink.PublishChunk(AudioChunk{Channel: channel, Target: target, CodecMIME: codecMIME(asset.MIME), Seq: seq, Data: append([]byte(nil), data[offset:end]...), Final: end == len(data), DurationMS: chunkDuration(asset.DurationMS, offset, end, len(data))})
		offset = end
	}
	return nil
}

// RouteSound publishes a generated or manifest sound using its requested
// channel, target, loop flag, and gain.
func (r *AudioRouter) RouteSound(effect domain.PlaySound, asset domain.Asset, data []byte) error {
	if err := r.RouteAsset(asset, data, effect.Channel, AudioTarget{Kind: effect.Target, Seat: effect.Seat}); err != nil {
		return err
	}
	r.sink.PublishMix(MixCommand{Channel: effect.Channel, Target: AudioTarget{Kind: effect.Target, Seat: effect.Seat}, Command: "play", TrackID: effect.Name, StartAtMS: int64(max(effect.DelayMS, 0)), Loop: effect.Loop, Gain: effect.Gain})
	return nil
}

// RouteMusicTransition publishes a bar-aligned crossfade command.
func (r *AudioRouter) RouteMusicTransition(cue MusicCue, target AudioTarget, gain, duck float32) error {
	if r == nil || r.sink == nil {
		return errors.New("audio router: router is not configured")
	}
	if cue.Kind == CueNone {
		return nil
	}
	command := "play"
	if cue.Kind == CueLoopTransition {
		command = "crossfade"
	}
	r.sink.PublishMix(MixCommand{Channel: vocab.SoundMusic, Target: target, Command: command, TrackID: cue.Track, StartAtMS: int64(cue.StartAtMS), DurationMS: cue.CrossfadeMS, Loop: cue.Kind == CueLoopTransition, Gain: gain, Duck: duck})
	return nil
}

// RouteMusicAsset sends a manifest-backed track and its mixer command in
// order, allowing the client to buffer bytes before starting at the cue.
func (r *AudioRouter) RouteMusicAsset(track domain.MusicTrack, data []byte, target AudioTarget, cue MusicCue) error {
	if err := r.RouteAsset(domain.Asset{ID: track.Asset, MIME: "audio/ogg;codecs=opus"}, data, vocab.SoundMusic, target); err != nil {
		return err
	}
	return r.RouteMusicTransition(cue, target, float32(track.Level), float32(track.Duck))
}

func codecMIME(mime string) string {
	if mime == "" {
		return "audio/ogg;codecs=opus"
	}
	return mime
}

func chunkDuration(total, offset, end, size int) int {
	if total <= 0 || size <= 0 {
		return 0
	}
	return total * (end - offset) / size
}
