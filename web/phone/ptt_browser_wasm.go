//go:build js && wasm

package phone

import (
	"errors"
	"sync"
	"syscall/js"
)

// BrowserRecorder wraps MediaRecorder without doing work in dataavailable.
type BrowserRecorder struct {
	recorder      js.Value
	blobs         chan js.Value
	callback      js.Func
	stopCallback  js.Func
	stopRequested chan struct{}
	done          chan struct{}
	failed        chan struct{}
	stopOnce      sync.Once
	failOnce      sync.Once
	errMu         sync.Mutex
	err           error
	disposeOnce   sync.Once
}

// NewBrowserRecorder creates a recorder on an already-open microphone stream.
func NewBrowserRecorder(stream js.Value, mimeType string, queue func([]byte) bool) (*BrowserRecorder, error) {
	if !stream.Truthy() || queue == nil {
		return nil, errors.New("media recorder stream and queue are required")
	}
	options := js.Global().Get("Object").New()
	options.Set("mimeType", mimeType)
	recorder := js.Global().Get("MediaRecorder").New(stream, options)
	b := &BrowserRecorder{recorder: recorder, blobs: make(chan js.Value, defaultPTTQueueSize), stopRequested: make(chan struct{}), done: make(chan struct{}), failed: make(chan struct{})}
	b.callback = js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		select {
		case b.blobs <- args[0].Get("data"):
		default:
			b.fail(errors.New("media recorder blob queue is full"))
		}
		return nil
	})
	b.stopCallback = js.FuncOf(func(js.Value, []js.Value) interface{} {
		b.stopOnce.Do(func() { close(b.stopRequested) })
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

// Done closes after the recorder has delivered and converted its final blob.
func (b *BrowserRecorder) Done() <-chan struct{} { return b.done }

// Err reports a recorder conversion or queue failure.
func (b *BrowserRecorder) Err() error {
	b.errMu.Lock()
	defer b.errMu.Unlock()
	return b.err
}

// Dispose removes browser listeners and releases their Go callbacks.
func (b *BrowserRecorder) Dispose() {
	if b == nil {
		return
	}
	b.disposeOnce.Do(func() {
		b.recorder.Call("removeEventListener", "dataavailable", b.callback)
		b.recorder.Call("removeEventListener", "stop", b.stopCallback)
		b.stopOnce.Do(func() { close(b.stopRequested) })
		b.callback.Release()
		b.stopCallback.Release()
	})
}

func (b *BrowserRecorder) readBlobs(queue func([]byte) bool) {
	defer close(b.done)
	for {
		select {
		case blob := <-b.blobs:
			if !b.convertBlob(blob, queue) {
				return
			}
		case <-b.stopRequested:
			for {
				select {
				case blob := <-b.blobs:
					if !b.convertBlob(blob, queue) {
						return
					}
				default:
					return
				}
			}
		}
	}
}

func (b *BrowserRecorder) convertBlob(blob js.Value, queue func([]byte) bool) bool {
	promise := blob.Call("arrayBuffer")
	converted := make(chan []byte, 1)
	rejected := make(chan struct{}, 1)
	then := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			converted <- nil
			return nil
		}
		view := js.Global().Get("Uint8Array").New(args[0])
		bytes := make([]byte, view.Get("byteLength").Int())
		js.CopyBytesToGo(bytes, view)
		converted <- bytes
		return nil
	})
	catch := js.FuncOf(func(js.Value, []js.Value) interface{} { rejected <- struct{}{}; return nil })
	promise.Call("then", then).Call("catch", catch)
	var bytes []byte
	select {
	case bytes = <-converted:
	case <-rejected:
		then.Release()
		catch.Release()
		b.fail(errors.New("media recorder blob conversion failed"))
		return false
	}
	then.Release()
	catch.Release()
	if len(bytes) == 0 || !queue(bytes) {
		b.fail(errors.New("media recorder chunk could not be queued"))
		return false
	}
	return true
}

func (b *BrowserRecorder) fail(err error) {
	b.errMu.Lock()
	if b.err == nil {
		b.err = err
	}
	b.errMu.Unlock()
	b.failOnce.Do(func() { close(b.failed); b.stopOnce.Do(func() { close(b.stopRequested) }) })
}
