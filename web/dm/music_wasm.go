//go:build js && wasm

package dm

import (
	"fmt"
	"math"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const musicFadeSeconds = 0.08

// MusicIndicator renders the operator-facing music state without exposing
// mixer controls on the presentation screen.
func MusicIndicator(model MusicModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		if !model.Active() {
			return html.Div(html.Props{Class: "df-dm-music df-dm-music-idle", Aria: map[string]string{"label": "Music idle"}, Style: map[string]string{"display": "none"}})
		}
		width := fmt.Sprintf("%.0f%%", model.DisplayLevel()*100)
		label := "Music"
		if model.Cue != "" {
			label += " · " + model.Cue
		}
		return html.Div(html.Props{Class: "df-dm-music", Role: "status", Aria: map[string]string{"label": label}, Style: map[string]string{"position": "absolute", "right": "1.25rem", "bottom": "1.25rem", "display": "inline-flex", "align-items": "center", "gap": "0.55rem", "padding": "0.45rem 0.7rem", "border": "1px solid rgba(217,164,65,.55)", "border-radius": "999px", "background": "rgba(15,17,23,.78)", "color": "#efe6d2", "font-size": "0.8rem", "letter-spacing": "0.08em", "text-transform": "uppercase"}},
			html.Span(html.Props{Class: "df-dm-music-glyph", Aria: map[string]string{"hidden": "true"}}, ui.Text("♫")),
			html.Span(html.Props{Class: "df-dm-music-label"}, ui.Text(label)),
			html.Span(html.Props{Class: "df-dm-music-meter", Style: map[string]string{"display": "inline-block", "width": "2.5rem", "height": "0.2rem", "border-radius": "999px", "background": "linear-gradient(90deg, #d9a441 " + width + ", rgba(239,230,210,.2) " + width + ")"}, Aria: map[string]string{"hidden": "true"}}),
		)
	}
}

// MusicPlayer plays decoded music buffers with loop points and bar-aligned
// transitions. It is intended to be owned by the DM screen.
type MusicPlayer struct {
	context     js.Value
	bus         js.Value
	current     MusicModel
	activeLevel float32
	startedAt   float64
	source      js.Value
	gain        js.Value
	loaded      map[string]js.Value
}

// NewMusicPlayer creates a player attached to the browser's AudioContext.
func NewMusicPlayer() *MusicPlayer {
	context := js.Global().Get("AudioContext").New()
	bus := context.Call("createGain")
	bus.Call("connect", context.Get("destination"))
	return &MusicPlayer{context: context, bus: bus, loaded: make(map[string]js.Value)}
}

// Apply follows a MusicView projection. A changed track loads asynchronously,
// then starts on the outgoing track's next bar boundary.
func (p *MusicPlayer) Apply(next MusicModel, elapsedMS int64) error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("dm: nil music player")
	}
	if next.TrackID == "" || next.URL == "" {
		p.Stop()
		return nil
	}
	if next.TrackID == p.current.TrackID && next.URL == p.current.URL {
		p.setGain(p.gain, next.Level)
		p.current = next
		p.activeLevel = next.Level
		return nil
	}
	if p.startedAt > 0 {
		elapsedMS = max64(0, int64(math.Round((p.context.Get("currentTime").Float()-p.startedAt)*1000)))
	}
	transition := PlanMusicTransition(p.current, next, elapsedMS)
	p.current = next
	if buffer, ok := p.loaded[next.URL]; ok {
		p.schedule(buffer, next, transition)
		return nil
	}
	request := js.Global().Get("fetch").Invoke(next.URL)
	var then js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer then.Release()
		if len(args) == 0 {
			return nil
		}
		return args[0].Call("arrayBuffer")
	})
	request = request.Call("then", then)
	var decode js.Func
	decode = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer decode.Release()
		if len(args) == 0 {
			return nil
		}
		return p.context.Call("decodeAudioData", args[0])
	})
	request = request.Call("then", decode)
	var ready js.Func
	ready = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer ready.Release()
		if len(args) > 0 && p.current.TrackID == next.TrackID && p.current.URL == next.URL {
			p.loaded[next.URL] = args[0]
			p.schedule(args[0], next, transition)
		}
		return nil
	})
	request.Call("then", ready)
	return nil
}

// Stop stops the active music source and clears the current track.
func (p *MusicPlayer) Stop() {
	if p == nil {
		return
	}
	if p.source.Truthy() {
		p.source.Call("stop")
	}
	p.source = js.Undefined()
	p.current = MusicModel{}
	p.activeLevel = 0
	p.startedAt = 0
}

// Close stops playback and releases the browser audio context.
func (p *MusicPlayer) Close() {
	if p == nil {
		return
	}
	p.Stop()
	if p.context.Truthy() {
		p.context.Call("close")
	}
	p.context = js.Undefined()
}

// Resume unlocks the browser audio context after a user gesture.
func (p *MusicPlayer) Resume() error {
	if p == nil || !p.context.Truthy() {
		return fmt.Errorf("dm: nil music player")
	}
	if promise := p.context.Call("resume"); promise.Truthy() {
		return nil
	}
	return fmt.Errorf("dm: audio context could not resume")
}

func (p *MusicPlayer) schedule(buffer js.Value, model MusicModel, transition MusicTransition) {
	oldSource, oldGain := p.source, p.gain
	oldLevel := p.activeLevel
	source := p.context.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Set("loop", true)
	source.Set("loopStart", float64(model.LoopStartMS)/1000)
	source.Set("loopEnd", float64(model.LoopEndMS)/1000)
	gain := p.context.Call("createGain")
	gain.Call("connect", p.bus)
	start := p.context.Get("currentTime").Float() + float64(max64(0, transition.DelayMS))/1000
	source.Call("connect", gain)
	source.Call("start", start)
	if oldSource.Truthy() && transition.Crossfade {
		end := start + float64(transition.DurationMS)/1000
		oldGain.Get("gain").Call("setValueCurveAtTime", equalPowerCurve(oldLevel, false), start, end-start)
		gain.Get("gain").Call("setValueCurveAtTime", equalPowerCurve(model.Level, true), start, end-start)
		oldSource.Call("stop", end)
	} else if oldSource.Truthy() {
		p.setGain(gain, 0)
		gain.Get("gain").Call("linearRampToValueAtTime", model.Level, start+musicFadeSeconds)
		oldSource.Call("stop", start)
	} else {
		p.setGain(gain, 0)
		gain.Get("gain").Call("linearRampToValueAtTime", model.Level, start+musicFadeSeconds)
	}
	p.source, p.gain = source, gain
	p.activeLevel = model.Level
	p.startedAt = start
}

func (p *MusicPlayer) setGain(gain js.Value, level float32) {
	if gain.Truthy() {
		gain.Get("gain").Set("value", level)
	}
}

func equalPowerCurve(level float32, fadeIn bool) js.Value {
	const samples = 32
	curve := js.Global().Get("Float32Array").New(samples)
	for i := 0; i < samples; i++ {
		progress := float64(i) / float64(samples-1)
		value := math.Cos(progress * math.Pi / 2)
		if fadeIn {
			value = math.Sin(progress * math.Pi / 2)
		}
		value *= float64(level)
		curve.SetIndex(i, value)
	}
	return curve
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
