//go:build js && wasm

package audio

import (
	"fmt"
	"syscall/js"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// Handle routes a Listen message through the mixer's channel graph.
// Decoding is promise-based so a browser callback never waits on audio work.
func (p *Player) Handle(message *dungeonfluxv1.AudioMessage) error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("audio: player is unavailable")
	}
	if message == nil {
		return fmt.Errorf("audio: mixer message is required")
	}
	if frame := message.GetFrame(); frame != nil {
		if message.GetChannel() == dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_VOICE {
			p.DuckVoice(!frame.GetFinal())
		}
		chunk, ok, err := p.scheduler.Accept(frame)
		if err != nil {
			return err
		}
		if ok {
			return p.Play(chunk)
		}
		return nil
	}
	if cancel := message.GetCancel(); cancel != nil {
		if cancel.GetAll() {
			p.Cancel("")
		} else {
			p.Cancel(cancel.GetUtteranceId())
		}
		return nil
	}
	if mix := message.GetMix(); mix != nil {
		return p.applyMix(mix, channelFor(message.GetChannel()))
	}
	if chunk := message.GetChunk(); chunk != nil {
		channel := channelFor(message.GetChannel())
		p.pending[channel] = append(p.pending[channel], encodedChunk{data: append([]byte(nil), chunk.GetData()...)})
		if chunk.GetFinal() {
			return p.flushPending(channel)
		}
	}
	return nil
}

func channelFor(channel dungeonfluxv1.AudioChannel) Channel {
	switch channel {
	case dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_MUSIC:
		return MusicChannel
	case dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_AMBIENCE:
		return AmbienceChannel
	case dungeonfluxv1.AudioChannel_AUDIO_CHANNEL_SFX:
		return SFXChannel
	default:
		return VoiceChannel
	}
}

func (p *Player) applyMix(command *dungeonfluxv1.AudioMixCommand, channel Channel) error {
	if command.GetTrackId() == "" {
		return fmt.Errorf("audio: mix command has no track id")
	}
	id := command.GetTrackId()
	if channel == SFXChannel {
		at := time.Duration(p.context.Get("currentTime").Float() * float64(time.Second))
		if !p.sfxGate.Allow(id, at) {
			p.discardPending(channel)
			return nil
		}
	}
	switch command.GetKind() {
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_STOP:
		p.StopTrack(id, int(command.GetDurationMs()))
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY,
		dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_CROSSFADE:
		if p.HasTrack(id) {
			p.discardPending(channel)
			return nil
		}
		p.reserved[id] = true
		if len(p.pending[channel]) == 0 {
			p.pendingCommands[channel] = pendingMix{command: command}
			return nil
		}
		return p.startPending(channel, command)
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_DUCK:
		duck := float64(command.GetDuck())
		if duck <= 0 {
			duck = float64(DuckGain)
		}
		p.bus(channel).Get("gain").Call("setTargetAtTime", duck, p.context.Get("currentTime"), .08)
	default:
		return fmt.Errorf("audio: unsupported mix command")
	}
	return nil
}

func (p *Player) flushPending(channel Channel) error {
	command, ok := p.pendingCommands[channel]
	if !ok {
		return nil
	}
	delete(p.pendingCommands, channel)
	return p.startPending(channel, command.command)
}

func (p *Player) startPending(channel Channel, command *dungeonfluxv1.AudioMixCommand) error {
	var size int
	for _, chunk := range p.pending[channel] {
		size += len(chunk.data)
	}
	data := make([]byte, 0, size)
	for _, chunk := range p.pending[channel] {
		data = append(data, chunk.data...)
	}
	delete(p.pending, channel)
	return p.decodeEncoded(command.GetTrackId(), channel, data, command.GetLoop(), float64(command.GetGain()), command.GetStartAtMs(), int(command.GetDurationMs()))
}

func (p *Player) discardPending(channel Channel) {
	delete(p.pending, channel)
	delete(p.pendingCommands, channel)
}

func (p *Player) decodeEncoded(id string, channel Channel, data []byte, loop bool, gain float64, delayMS int64, fadeInMS int) error {
	if len(data) == 0 {
		return fmt.Errorf("audio: encoded chunk is empty")
	}
	bytes := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bytes, data)
	promise := p.context.Call("decodeAudioData", bytes.Get("buffer"))
	// The callbacks run when decoding settles, so they are released there.
	// Releasing them right after registering (as before) made the browser
	// call released functions: every streamed music, ambience, and stinger
	// track decoded and was never started, and its reservation stuck.
	var callback, rejected js.Func
	release := func() {
		callback.Release()
		rejected.Release()
	}
	callback = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer release()
		if len(args) == 0 || !p.reserved[id] {
			return nil
		}
		p.startDecoded(id, channel, args[0], loop, gain, delayMS, fadeInMS)
		return nil
	})
	rejected = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		defer release()
		delete(p.reserved, id)
		return nil
	})
	promise.Call("then", callback, rejected)
	return nil
}

func (p *Player) startDecoded(id string, channel Channel, buffer js.Value, loop bool, gain float64, delayMS int64, fadeInMS int) {
	source := p.context.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Set("loop", loop)
	trackGain := p.context.Call("createGain")
	trackGain.Call("connect", p.bus(channel))
	source.Call("connect", trackGain)
	if loop && (channel == MusicChannel || channel == AmbienceChannel) {
		// One bed per channel: a new phase loop crossfades out the old one.
		if p.replaceBed(id, channel) {
			fadeInMS = max(fadeInMS, bedCrossfadeMS)
		}
	}
	when := p.context.Get("currentTime").Float() + float64(max64(delayMS, 0))/1000
	trackGain.Get("gain").Call("setValueAtTime", 0, when)
	trackGain.Get("gain").Call("linearRampToValueAtTime", gain, when+float64(max(fadeInMS, 0))/1000)
	if channel == SFXChannel && !loop {
		// A one-shot is not a held track: clear the reservation so the same
		// effect can play again (the cue gate already prevents stacking).
		delete(p.reserved, id)
		source.Call("start", when)
		return
	}
	p.tracks[id] = append(p.tracks[id], source)
	p.trackGains[id] = append(p.trackGains[id], trackGain)
	source.Call("start", when)
	if !loop {
		p.played[id] = true
		var goDelete js.Func
		goDelete = js.FuncOf(func(_ js.Value, _ []js.Value) any {
			delete(p.tracks, id)
			delete(p.trackGains, id)
			delete(p.reserved, id)
			goDelete.Release()
			return nil
		})
		durationMS := int(buffer.Get("duration").Float() * 1000)
		js.Global().Get("setTimeout").Invoke(goDelete, durationMS+100)
	}
}

// bedCrossfadeMS is the crossfade between two looping beds on one channel.
const bedCrossfadeMS = 1200

// replaceBed records id as the channel's looping bed and fades out any other
// bed on that channel. It reports whether a previous bed was replaced.
func (p *Player) replaceBed(id string, channel Channel) bool {
	if p.beds == nil {
		p.beds = make(map[Channel]string)
	}
	previous, ok := p.beds[channel]
	p.beds[channel] = id
	if !ok || previous == id {
		return false
	}
	p.StopTrack(previous, bedCrossfadeMS)
	return true
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}

// DuckVoice lowers music and ambience before narration and restores them after it.
func (p *Player) DuckVoice(active bool) {
	target := float64(DuckGain)
	tau := .08
	if !active {
		target = 1
		tau = .35
	}
	for _, channel := range []Channel{MusicChannel, AmbienceChannel} {
		p.bus(channel).Get("gain").Call("setTargetAtTime", target, p.context.Get("currentTime"), tau)
	}
}
