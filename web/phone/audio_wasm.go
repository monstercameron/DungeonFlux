//go:build js && wasm

package phone

import (
	"context"
	"errors"
	"syscall/js"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// ConfigureAudio connects phone effects to the generated AudioService client.
// It does not open the stream; callers invoke UnlockAudio from a user gesture.
func (a *PhoneAudio) ConfigureAudio(service phoneAudioService, seatToken string, seat int32) {
	if a == nil {
		return
	}
	a.service, a.seatToken, a.seat = service, seatToken, seat
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
}

func (a *PhoneAudio) receiveAudio(ctx context.Context, player *phoneAudioPlayer) {
	defer player.close()
	stream, err := a.service.Listen(ctx, &df.ListenRequest{SeatToken: a.seatToken})
	if err != nil {
		return
	}
	for ctx.Err() == nil {
		message, recvErr := stream.Recv()
		if recvErr != nil {
			return
		}
		if !a.Enqueue(message, a.seat) {
			continue
		}
		for {
			item, ok := a.Next()
			if !ok {
				break
			}
			if err := player.play(item, a.HapticsEnabled()); err != nil {
				break
			}
		}
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

func (p *phoneAudioPlayer) play(item PhoneAudioMessage, vibrate bool) error {
	chunk := item.Message.GetChunk()
	if chunk == nil || len(chunk.GetData()) == 0 {
		return errors.New("phone audio: empty effect")
	}
	bytes := js.Global().Get("Uint8Array").New(len(chunk.GetData()))
	for index, value := range chunk.GetData() {
		bytes.SetIndex(index, value)
	}
	promise := p.context.Call("decodeAudioData", bytes.Get("buffer"))
	then := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		source := p.context.Call("createBufferSource")
		source.Set("buffer", args[0])
		source.Call("connect", p.bus)
		source.Call("start")
		if vibrate {
			navigator := js.Global().Get("navigator")
			if navigator.Truthy() && navigator.Get("vibrate").Truthy() {
				navigator.Call("vibrate", 18)
			}
		}
		return nil
	})
	reject := js.FuncOf(func(js.Value, []js.Value) interface{} { return nil })
	promise.Call("then", then).Call("catch", reject)
	then.Release()
	reject.Release()
	return nil
}

func (p *phoneAudioPlayer) close() {
	if p != nil && p.context.Truthy() {
		p.context.Call("close")
	}
}
