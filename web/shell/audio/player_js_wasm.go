//go:build js && wasm

package audio

import (
	"fmt"
	"syscall/js"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// Player schedules PCM chunks on a browser AudioContext and tracks sources by
// utterance so AudioCancel can stop them without waiting for playback to end.
type Player struct {
	context         js.Value
	sources         map[string][]js.Value
	buses           map[Channel]js.Value
	tracks          map[string][]js.Value
	trackGains      map[string][]js.Value
	reserved        map[string]bool
	played          map[string]bool
	pending         map[Channel][]encodedChunk
	pendingCommands map[Channel]pendingMix
	scheduler       Scheduler
}

type encodedChunk struct {
	data []byte
}

type pendingMix struct {
	command *dungeonfluxv1.AudioMixCommand
}

// NewPlayer creates a player using a new browser AudioContext.
func NewPlayer() *Player {
	context := js.Global().Get("AudioContext").New()
	buses := make(map[Channel]js.Value)
	for _, channel := range []Channel{VoiceChannel, MusicChannel, AmbienceChannel, SFXChannel} {
		gain := context.Call("createGain")
		gain.Get("gain").Set("value", 1)
		gain.Call("connect", context.Get("destination"))
		buses[channel] = gain
	}
	return &Player{context: context, sources: make(map[string][]js.Value), buses: buses, tracks: make(map[string][]js.Value), trackGains: make(map[string][]js.Value), reserved: make(map[string]bool), played: make(map[string]bool), pending: make(map[Channel][]encodedChunk), pendingCommands: make(map[Channel]pendingMix)}
}

// Resume unlocks PCM playback from the table's explicit user gesture.
func (p *Player) Resume() error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("audio: player is unavailable")
	}
	p.context.Call("resume")
	return nil
}

// Close stops scheduled audio and releases the browser audio context.
func (p *Player) Close() {
	if p == nil || !p.context.Truthy() {
		return
	}
	p.Cancel("")
	p.context.Call("close")
	p.context = js.Undefined()
}

// Play schedules one PCM chunk according to its plan.
func (p *Player) Play(chunk ScheduledChunk) error {
	if chunk.SampleRate <= 0 || len(chunk.PCM)%bytesPerSample != 0 {
		return fmt.Errorf("audio: invalid scheduled chunk")
	}
	frames := len(chunk.PCM) / bytesPerSample
	buffer := p.context.Call("createBuffer", 1, frames, chunk.SampleRate)
	data := js.Global().Get("Float32Array").New(frames)
	for i := 0; i < frames; i++ {
		sample := int16(uint16(chunk.PCM[i*2]) | uint16(chunk.PCM[i*2+1])<<8)
		data.SetIndex(i, float32(sample)/32768)
	}
	buffer.Call("copyToChannel", data, 0)
	source := p.context.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Call("connect", p.bus(VoiceChannel))
	when := p.context.Get("currentTime").Float() + chunk.Start.Seconds()
	source.Call("start", when)
	p.sources[chunk.UtteranceID] = append(p.sources[chunk.UtteranceID], source)
	return nil
}

func (p *Player) bus(channel Channel) js.Value {
	if bus, ok := p.buses[channel]; ok {
		return bus
	}
	return p.context.Get("destination")
}

// HasTrack reports whether a track is playing or already reserved for decode.
func (p *Player) HasTrack(id string) bool {
	return p != nil && (p.reserved[id] || p.played[id] || len(p.tracks[id]) > 0)
}

// PlayURL decodes one gRPC-backed Blob URL into a tracked audio source.
// Non-Blob URLs are rejected so audio never falls back to an HTTP fetch.
func (p *Player) PlayURL(id string, channel Channel, url string, loop bool, gain float32, fadeInMS int) error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("audio: player is unavailable")
	}
	if id == "" || url == "" {
		return fmt.Errorf("audio: track id and URL are required")
	}
	if p.HasTrack(id) {
		return nil
	}
	if len(url) < 5 || url[:5] != "blob:" {
		return fmt.Errorf("audio: URL is not a gRPC-backed Blob URL")
	}
	p.reserved[id] = true
	request := js.Global().Get("fetch").Invoke(url)
	var then js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		return args[0].Call("arrayBuffer")
	})
	request = request.Call("then", then)
	var decode js.Func
	decode = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		return p.context.Call("decodeAudioData", args[0])
	})
	request = request.Call("then", decode)
	var ready js.Func
	ready = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer func() {
			then.Release()
			decode.Release()
			ready.Release()
		}()
		if len(args) > 0 && p.reserved[id] {
			p.startDecoded(id, channel, args[0], loop, float64(gain), 0, fadeInMS)
		}
		return nil
	})
	request.Call("then", ready)
	return nil
}

// StopTrack fades out and stops one music or ambience source.
func (p *Player) StopTrack(id string, fadeOutMS int) {
	if p == nil {
		return
	}
	p.cancelPending(id)
	when := p.context.Get("currentTime").Float()
	end := when + float64(max(0, fadeOutMS))/1000
	for index, source := range p.tracks[id] {
		if fadeOutMS > 0 && index < len(p.trackGains[id]) {
			gain := p.trackGains[id][index].Get("gain")
			gain.Call("setTargetAtTime", 0, when, .08)
		}
		if fadeOutMS > 0 {
			source.Call("stop", end)
		} else {
			source.Call("stop")
		}
	}
	delete(p.tracks, id)
	delete(p.trackGains, id)
	delete(p.reserved, id)
}

func (p *Player) cancelPending(id string) {
	for channel, pending := range p.pendingCommands {
		if pending.command != nil && pending.command.GetTrackId() == id {
			delete(p.pendingCommands, channel)
			delete(p.pending, channel)
		}
	}
}

// Cancel stops every source for utteranceID, or all tracked sources when the
// ID is empty.
func (p *Player) Cancel(utteranceID string) {
	if utteranceID == "" {
		for id := range p.sources {
			p.Cancel(id)
		}
		return
	}
	for _, source := range p.sources[utteranceID] {
		source.Call("stop")
	}
	delete(p.sources, utteranceID)
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
