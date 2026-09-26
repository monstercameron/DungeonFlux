//go:build js && wasm

package phone

import (
	"errors"
	"syscall/js"
)

// BrowserRecorder wraps MediaRecorder without doing work in dataavailable.
type BrowserRecorder struct {
	recorder     js.Value
	blobs        chan js.Value
	callback     js.Func
	stopCallback js.Func
	closed       chan struct{}
}

// NewBrowserRecorder creates a recorder on an already-open microphone stream.
func NewBrowserRecorder(stream js.Value, mimeType string, queue func([]byte) bool) (*BrowserRecorder, error) {
	if !stream.Truthy() || queue == nil {
		return nil, errors.New("media recorder stream and queue are required")
	}
	options := js.Global().Get("Object").New()
	options.Set("mimeType", mimeType)
	recorder := js.Global().Get("MediaRecorder").New(stream, options)
	b := &BrowserRecorder{recorder: recorder, blobs: make(chan js.Value, defaultPTTQueueSize), closed: make(chan struct{})}
	b.callback = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		select {
		case b.blobs <- args[0].Get("data"):
		default:
			// The uploader is deliberately bounded; a full queue fails closed.
		}
		return nil
	})
	b.stopCallback = js.FuncOf(func(js.Value, []js.Value) interface{} {
		select {
		case <-b.closed:
		default:
			close(b.closed)
		}
		return nil
	})
	recorder.Call("addEventListener", "dataavailable", b.callback)
	recorder.Call("addEventListener", "stop", b.stopCallback)
	go b.readBlobs(queue)
	return b, nil
}

// Start begins 100 ms MediaRecorder timeslices.
func (b *BrowserRecorder) Start() error {
	if b == nil || !b.recorder.Truthy() {
		return errors.New("media recorder is unavailable")
	}
	b.recorder.Call("start", 100)
	return nil
}

// Stop ends the MediaRecorder; its final dataavailable event is queued too.
func (b *BrowserRecorder) Stop() error {
	if b == nil || !b.recorder.Truthy() {
		return errors.New("media recorder is unavailable")
	}
	b.recorder.Call("stop")
	return nil
}

func (b *BrowserRecorder) readBlobs(queue func([]byte) bool) {
	for {
		select {
		case blob := <-b.blobs:
			promise := blob.Call("arrayBuffer")
			then := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
				if len(args) == 0 {
					return nil
				}
				view := js.Global().Get("Uint8Array").New(args[0])
				bytes := make([]byte, view.Get("byteLength").Int())
				js.CopyBytesToGo(bytes, view)
				queue(bytes)
				return nil
			})
			promise.Call("then", then)
		case <-b.closed:
			return
		}
	}
}
