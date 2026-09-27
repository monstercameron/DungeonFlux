//go:build js && wasm

package audio

import (
	"fmt"
	"syscall/js"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// Player schedules PCM chunks on a browser AudioContext and tracks sources by
// utterance so AudioCancel can stop them without waiting for playback to end.
type Player struct {
	context    js.Value
	sources    map[string][]js.Value
	buses      map[Channel]js.Value
	tracks     map[string][]js.Value
	trackGains map[string][]js.Value
	reserved   map[string]bool
	// bedLevels holds each bed's authored gain so a stinger can dip the
	// bed under it and restore it afterwards.
	bedLevels       map[string]float64
	pending         map[Channel][]encodedChunk
	pendingCommands map[Channel]pendingMix
	scheduler       Scheduler
	sfxGate         CueGate
	// lines anchors each voice line's chunks to one AudioContext base time
	// and tracks when its scheduled audio ends.
	lines lineClock
	// lineGains holds one gain node per voice line so an interrupted line
	// fades out instead of cutting mid-word.
	lineGains map[string]js.Value
	// beds names the looping track currently playing on each bed channel
	// (music, ambience) so a new phase bed crossfades the old one out.
	beds map[Channel]string
}

// voiceFadeSeconds is the fade applied when a line is cancelled mid-play.
const voiceFadeSeconds = 0.06

// PlaySFXURL plays a short manifest-backed effect without treating its asset
// ID as a permanently playing track. The cue gate prevents accidental stacks
// while allowing the same effect to be used again later in the scene.
func (p *Player) PlaySFXURL(id, url string, gain float32) error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("audio: player is unavailable")
	}
	if id == "" || url == "" || len(url) < 5 || url[:5] != "blob:" {
		return fmt.Errorf("audio: SFX id and Blob URL are required")
	}
	at := time.Duration(p.context.Get("currentTime").Float() * float64(time.Second))
	if !p.sfxGate.Allow(id, at) {
		return nil
	}
	request := js.Global().Get("fetch").Invoke(url)
	var then, decode, ready js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		return args[0].Call("arrayBuffer")
	})
	decode = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		return p.context.Call("decodeAudioData", args[0])
	})
	ready = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer func() { then.Release(); decode.Release(); ready.Release() }()
		if len(args) > 0 {
			p.startDecoded(id, SFXChannel, args[0], false, float64(gain), 0, 0)
		}
		return nil
	})
	request.Call("then", then).Call("then", decode).Call("then", ready)
	return nil
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
	return &Player{context: context, sources: make(map[string][]js.Value), buses: buses, tracks: make(map[string][]js.Value), trackGains: make(map[string][]js.Value), reserved: make(map[string]bool), bedLevels: make(map[string]float64), pending: make(map[Channel][]encodedChunk), pendingCommands: make(map[Channel]pendingMix)}
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
	now := p.context.Get("currentTime").Float()
	p.releaseFinished(now)
	source.Call("connect", p.lineGain(chunk.UtteranceID))
	// Chunk.Start is an offset from the line's first chunk; the line clock
	// anchors it to a per-line base time so chunks play back to back.
	when := p.lines.place(chunk.UtteranceID, chunk.Start.Seconds(), float64(frames)/float64(chunk.SampleRate), now)
	source.Call("start", when)
	p.sources[chunk.UtteranceID] = append(p.sources[chunk.UtteranceID], source)
	if chunk.Final {
		p.lines.finish(chunk.UtteranceID)
	}
	return nil
}

// lineGain returns the utterance's gain node, creating it on the voice bus.
func (p *Player) lineGain(id string) js.Value {
	if p.lineGains == nil {
		p.lineGains = make(map[string]js.Value)
	}
	if gain, ok := p.lineGains[id]; ok {
		return gain
	}
	gain := p.context.Call("createGain")
	gain.Get("gain").Set("value", 1)
	gain.Call("connect", p.bus(VoiceChannel))
	p.lineGains[id] = gain
	return gain
}

// releaseFinished drops the sources and gain nodes of lines that have played
// out, so a long session does not keep every chunk buffer alive.
func (p *Player) releaseFinished(now float64) {
	for _, id := range p.lines.expired(now) {
		if gain, ok := p.lineGains[id]; ok {
			gain.Call("disconnect")
			delete(p.lineGains, id)
		}
		delete(p.sources, id)
		p.lines.drop(id)
	}
}

func (p *Player) bus(channel Channel) js.Value {
	if bus, ok := p.buses[channel]; ok {
		return bus
	}
	return p.context.Get("destination")
}

// HasTrack reports whether a track is playing or already reserved for decode.
// A one-shot that has finished is not held, so a stinger plays again when a
// reset run reaches its moment a second time.
func (p *Player) HasTrack(id string) bool {
	return p != nil && (p.reserved[id] || len(p.tracks[id]) > 0)
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
	for channel, bed := range p.beds {
		if bed == id {
			delete(p.beds, channel)
		}
	}
	when := p.context.Get("currentTime").Float()
	end := when + float64(max(0, fadeOutMS))/1000
	tau := float64(max(fadeOutMS, 240)) / 3000
	for index, source := range p.tracks[id] {
		if fadeOutMS > 0 && index < len(p.trackGains[id]) {
			gain := p.trackGains[id][index].Get("gain")
			if gain.Get("cancelAndHoldAtTime").Truthy() {
				gain.Call("cancelAndHoldAtTime", when)
			} else {
				gain.Call("cancelScheduledValues", when)
			}
			gain.Call("setTargetAtTime", 0, when, tau)
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

// Cancel fades out and stops every source for utteranceID, or all tracked
// sources when the ID is empty. It is only reached for a line the server
// interrupted (AudioCancel) or when the player closes; a line that finished
// normally is never cancelled, so its queued tail always plays out.
func (p *Player) Cancel(utteranceID string) {
	if utteranceID == "" {
		for id := range p.sources {
			p.Cancel(id)
		}
		p.scheduler.Cancel("")
		return
	}
	p.scheduler.Cancel(utteranceID)
	if !p.context.Truthy() {
		delete(p.sources, utteranceID)
		delete(p.lineGains, utteranceID)
		p.lines.drop(utteranceID)
		return
	}
	now := p.context.Get("currentTime").Float()
	stopAt := now + 2*voiceFadeSeconds
	if gain, ok := p.lineGains[utteranceID]; ok {
		gain.Get("gain").Call("setTargetAtTime", 0, now, voiceFadeSeconds/3)
		delete(p.lineGains, utteranceID)
	}
	for _, source := range p.sources[utteranceID] {
		source.Call("stop", stopAt)
	}
	delete(p.sources, utteranceID)
	p.lines.drop(utteranceID)
}

func max(left, right int) int {
	if left > right {
		return left
	}
	return right
}
