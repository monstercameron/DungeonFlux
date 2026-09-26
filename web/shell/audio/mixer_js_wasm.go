//go:build js && wasm

package audio

import (
	"fmt"
	"syscall/js"

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
		return p.decodeEncoded(fmt.Sprintf("chunk-%d", chunk.GetSeq()), channelFor(message.GetChannel()), chunk.GetData(), false, 1)
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
	switch command.GetKind() {
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_STOP:
		p.stopTrack(id)
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_PLAY,
		dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_CROSSFADE:
		for _, source := range p.tracks[id] {
			source.Set("loop", command.GetLoop())
		}
	case dungeonfluxv1.AudioMixCommandKind_AUDIO_MIX_COMMAND_KIND_DUCK:
		duck := float64(command.GetDuck())
		if duck <= 0 {
			duck = float64(DuckGain)
		}
		p.bus(channel).Get("gain").Call("setTargetAtTime", duck, p.context.Get("currentTime"), .08)
	default:
		return fmt.Errorf("audio: unsupported mix command")
	}
	if channel == MusicChannel || channel == AmbienceChannel {
		p.setBusGain(channel, float64(command.GetGain()))
	}
	return nil
}

func (p *Player) setBusGain(channel Channel, gain float64) {
	if gain <= 0 {
		gain = 1
	}
	p.bus(channel).Get("gain").Call("setTargetAtTime", gain, p.context.Get("currentTime"), .08)
}

func (p *Player) stopTrack(id string) {
	for _, source := range p.tracks[id] {
		source.Call("stop")
	}
	delete(p.tracks, id)
}

func (p *Player) decodeEncoded(id string, channel Channel, data []byte, loop bool, gain float64) error {
	if len(data) == 0 {
		return fmt.Errorf("audio: encoded chunk is empty")
	}
	bytes := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bytes, data)
	promise := p.context.Call("decodeAudioData", bytes.Get("buffer"))
	callback := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		buffer := args[0]
		source := p.context.Call("createBufferSource")
		source.Set("buffer", buffer)
		source.Set("loop", loop)
		source.Call("connect", p.bus(channel))
		p.setBusGain(channel, gain)
		p.tracks[id] = append(p.tracks[id], source)
		source.Call("start", p.context.Get("currentTime"))
		return nil
	})
	rejected := js.FuncOf(func(_ js.Value, _ []js.Value) any { return nil })
	promise.Call("then", callback).Call("catch", rejected)
	callback.Release()
	rejected.Release()
	return nil
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
