//go:build js && wasm

package splat

import (
	"errors"
	"sync"
	"syscall/js"
)

// Bridge connects typed battlefield messages to window.dfSplat.
type Bridge struct {
	mu       sync.Mutex
	module   js.Value
	callback js.Func
	events   chan Event
	closed   bool
}

// New creates a browser bridge with a bounded event channel.
func New(buffer int) *Bridge {
	if buffer < 1 {
		buffer = 1
	}
	b := &Bridge{module: js.Global().Get("dfSplat"), events: make(chan Event, buffer)}
	b.callback = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		b.push(args[0].String())
		return nil
	})
	if b.module.Truthy() {
		b.module.Call("onEvent", b.callback)
	}
	return b
}

func (b *Bridge) push(raw string) {
	event, err := decodeEvent(raw)
	if err != nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	select {
	case b.events <- event:
	default:
		// Stats are periodic and can be dropped; terminal events replace the oldest.
		select {
		case <-b.events:
		default:
		}
		select {
		case b.events <- event:
		default:
		}
	}
}

func (b *Bridge) send(kind string, value any) error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return errors.New("splat bridge is closed")
	}
	module := b.module
	b.mu.Unlock()
	if !module.Truthy() {
		return errors.New("window.dfSplat is unavailable")
	}
	payload, err := envelope(kind, value)
	if err != nil {
		return err
	}
	module.Call("send", string(payload))
	return nil
}

// Init loads the battlefield scene.
func (b *Bridge) Init(value Init) error { return b.send("init", value) }

// Scene sends a full battlefield snapshot.
func (b *Bridge) Scene(value Scene) error { return b.send("scene", value) }

// Effects sends a cinematic effects command to the browser.
func (b *Bridge) Effects(value Effects) error { return b.send("effects", value) }

// Pause freezes or resumes browser rendering.
func (b *Bridge) Pause(value Pause) error { return b.send("pause", value) }

// Dispose unloads the PlayCanvas scene.
func (b *Bridge) Dispose() error { return b.send("dispose", nil) }

// Events returns the bounded stream of JavaScript events.
func (b *Bridge) Events() <-chan Event { return b.events }

// Close releases the JavaScript callback and closes the event stream.
func (b *Bridge) Close() {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return
	}
	b.closed = true
	b.callback.Release()
	close(b.events)
	b.mu.Unlock()
}
