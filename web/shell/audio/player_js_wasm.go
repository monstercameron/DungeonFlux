//go:build js && wasm

package audio

import (
	"fmt"
	"syscall/js"
)

// Player schedules PCM chunks on a browser AudioContext and tracks sources by
// utterance so AudioCancel can stop them without waiting for playback to end.
type Player struct {
	context js.Value
	sources map[string][]js.Value
}

// NewPlayer creates a player using a new browser AudioContext.
func NewPlayer() *Player {
	return &Player{context: js.Global().Get("AudioContext").New(), sources: make(map[string][]js.Value)}
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
	source.Call("connect", p.context.Get("destination"))
	when := p.context.Get("currentTime").Float() + chunk.Start.Seconds()
	source.Call("start", when)
	p.sources[chunk.UtteranceID] = append(p.sources[chunk.UtteranceID], source)
	return nil
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
