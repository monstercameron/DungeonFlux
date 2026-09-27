//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"syscall/js"
	"time"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ConfigureAudio connects phone effects to the generated AudioService client.
// It does not open the stream; callers invoke UnlockAudio from a user gesture.
func (a *PhoneAudio) ConfigureAudio(service phoneAudioService, seatToken string, seat int32) {
	if a == nil {
		return
	}
	a.service, a.seatToken, a.seat = service, seatToken, seat
	a.installTapListener()
	mediaQuery := js.Global().Call("matchMedia", "(prefers-reduced-motion: reduce)")
	a.SetReducedMotion(mediaQuery.Get("matches").Truthy())
}

// UnlockAudio resumes the Web Audio context and starts the seat stream. It is
// safe to call repeatedly; only the first successful tap opens Listen.
func (a *PhoneAudio) UnlockAudio(ctx context.Context) error {
	if a == nil || a.service == nil {
		return errors.New("phone audio: service is unavailable")
	}
	if a.ctx != nil {
		return nil
	}
	a.ctx, a.cancel = context.WithCancel(ctx)
	contextValue := js.Global().Get("AudioContext")
	if !contextValue.Truthy() {
		return errors.New("phone audio: Web Audio is unavailable")
	}
	player := newPhoneAudioPlayer(contextValue.New())
	a.player = player
	if err := player.resume(); err != nil {
		a.cancel()
		a.ctx, a.cancel = nil, nil
		return err
	}
	go a.receiveAudio(a.ctx, player)
	return nil
}

// CloseAudio stops the listener and releases browser resources.
func (a *PhoneAudio) CloseAudio() {
	if a == nil {
		return
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.ctx, a.cancel = nil, nil
	a.player = nil
}

func (a *PhoneAudio) installTapListener() {
	if a == nil || a.tapInstalled {
		return
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	a.tapInstalled = true
	callback := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		// Any tap is a user gesture, so it may start Web Audio. Waiting for the
		// small "Enable sound" pill meant players never heard their tap, join,
		// ready or dice sounds (or their seat's voice and SFX stream).
		if a.player == nil && a.service != nil {
			_ = a.UnlockAudio(context.Background())
		}
		if a.player == nil {
			return nil
		}
		target := args[0]
		if target.Truthy() && target.Get("closest").Truthy() {
			target = target.Call("closest", "button")
		}
		if !target.Truthy() {
			return nil
		}
		cue := TapCue(target.Get("textContent").String())
		at := time.Duration(js.Global().Get("performance").Call("now").Float() * float64(time.Millisecond))
		if !a.AllowLocalCue(cue, at) {
			return nil
		}
		if asset := localCueAsset(cue); asset != "" {
			if url := ArtURL(asset); url != "" {
				_ = a.player.playURL(asset, url)
			}
		}
		return nil
	})
	document.Call("addEventListener", "click", callback)
}

func (a *PhoneAudio) receiveAudio(ctx context.Context, player *phoneAudioPlayer) {
	defer player.close()
	stream, err := a.service.Listen(ctx, &df.ListenRequest{SeatToken: a.seatToken})
	if err != nil {
		return
	}
	var assembler sfxAssembler
	for ctx.Err() == nil {
		message, recvErr := stream.Recv()
		if recvErr != nil {
			return
		}
		effect, ok := assembler.Accept(message, a.seat)
		if !ok || a.queue.Muted() {
			continue
		}
		_ = player.play(effect, a.HapticsEnabled())
	}
}

type phoneAudioPlayer struct {
	context js.Value
	bus     js.Value
}

func newPhoneAudioPlayer(context js.Value) *phoneAudioPlayer {
	gain := context.Call("createGain")
	gain.Get("gain").Set("value", 0.8)
	gain.Call("connect", context.Get("destination"))
	return &phoneAudioPlayer{context: context, bus: gain}
}

func (p *phoneAudioPlayer) resume() error {
	if p == nil || !p.context.Truthy() {
		return errors.New("phone audio: context is unavailable")
	}
	p.context.Call("resume")
	return nil
}

func (p *phoneAudioPlayer) playURL(id, url string) error {
	if p == nil || !p.context.Truthy() || id == "" || url == "" {
		return errors.New("phone audio: local cue is unavailable")
	}
	request := js.Global().Get("fetch").Invoke(url)
	var then, decode, ready js.Func
	then = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		return args[0].Call("arrayBuffer")
	})
	decode = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		return p.context.Call("decodeAudioData", args[0])
	})
	ready = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		defer func() { then.Release(); decode.Release(); ready.Release() }()
		if len(args) == 0 {
			return nil
		}
		source := p.context.Call("createBufferSource")
		source.Set("buffer", args[0])
		source.Call("connect", p.bus)
		source.Call("start")
		return nil
	})
	request.Call("then", then).Call("then", decode).Call("then", ready)
	return nil
}

// play decodes one streamed effect and starts it after its delay. The
// decode callbacks are released when decoding settles: releasing them right
// after registering (as before) left every pushed phone effect silent.
func (p *phoneAudioPlayer) play(effect StreamedSFX, vibrate bool) error {
	if p == nil || !p.context.Truthy() || len(effect.Data) == 0 {
		return errors.New("phone audio: empty effect")
	}
	bytes := js.Global().Get("Uint8Array").New(len(effect.Data))
	js.CopyBytesToJS(bytes, effect.Data)
	promise := p.context.Call("decodeAudioData", bytes.Get("buffer"))
	var then, reject js.Func
	release := func() { then.Release(); reject.Release() }
	then = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		defer release()
		if len(args) == 0 {
			return nil
		}
		gain := p.context.Call("createGain")
		gain.Get("gain").Set("value", float64(effect.Gain))
		gain.Call("connect", p.bus)
		source := p.context.Call("createBufferSource")
		source.Set("buffer", args[0])
		source.Call("connect", gain)
		source.Call("start", p.context.Get("currentTime").Float()+float64(effect.DelayMS)/1000)
		if vibrate {
			p.vibrateAfter(effect.DelayMS)
		}
		return nil
	})
	reject = js.FuncOf(func(js.Value, []js.Value) interface{} {
		release()
		return nil
	})
	promise.Call("then", then, reject)
	return nil
}

func (p *phoneAudioPlayer) vibrateAfter(delayMS int64) {
	navigator := js.Global().Get("navigator")
	if !navigator.Truthy() || !navigator.Get("vibrate").Truthy() {
		return
	}
	var buzz js.Func
	buzz = js.FuncOf(func(js.Value, []js.Value) interface{} {
		buzz.Release()
		navigator.Call("vibrate", 18)
		return nil
	})
	js.Global().Call("setTimeout", buzz, delayMS)
}

func (p *phoneAudioPlayer) close() {
	if p != nil && p.context.Truthy() {
		p.context.Call("close")
	}
}
