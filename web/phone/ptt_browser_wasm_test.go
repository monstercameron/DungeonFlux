//go:build js && wasm

package phone

import (
	"strings"
	"syscall/js"
	"testing"
	"time"
)

func installRecorderFake() func() {
	js.Global().Call("eval", `
(function(){
function DFRecorder(){this.h={};}
DFRecorder.prototype.addEventListener=function(n,f){this.h[n]=f;};
DFRecorder.prototype.removeEventListener=function(n){delete this.h[n];};
DFRecorder.prototype.start=function(){if(globalThis.DFThrowStart)throw new Error("start");};
DFRecorder.prototype.stop=function(){if(globalThis.DFThrowStop)throw new Error("stop");if(this.h.dataavailable)this.h.dataavailable({data:{arrayBuffer:function(){return Promise.resolve(new Uint8Array([1,2,3]).buffer);}}});if(this.h.stop)this.h.stop();};
DFRecorder.prototype.emit=function(){if(this.h.dataavailable)this.h.dataavailable({data:{arrayBuffer:function(){return Promise.resolve(new Uint8Array([4]).buffer);}}});};
DFRecorder.prototype.emitReject=function(){if(this.h.dataavailable)this.h.dataavailable({data:{arrayBuffer:function(){return Promise.reject(new Error("bad blob"));}}});};
DFRecorder.isTypeSupported=function(){return true;};
globalThis.MediaRecorder=DFRecorder;})()`)
	return func() {
		js.Global().Call("eval", "delete globalThis.MediaRecorder; delete globalThis.DFThrowStart; delete globalThis.DFThrowStop")
	}
}

func waitDone(t *testing.T, recorder *BrowserRecorder) {
	t.Helper()
	select {
	case <-recorder.Done():
	case <-time.After(time.Second):
		t.Fatal("recorder did not finish")
	}
}

func TestBrowserRecorder_DrainsFinalBlobInOrder(t *testing.T) {
	cleanup := installRecorderFake()
	defer cleanup()
	stream := js.Global().Get("Object").New()
	var chunks [][]byte
	recorder, err := NewBrowserRecorder(stream, "audio/webm", func(chunk []byte) bool { chunks = append(chunks, chunk); return true })
	if err != nil {
		t.Fatal(err)
	}
	if err := recorder.Start(); err != nil {
		t.Fatal(err)
	}
	recorder.recorder.Call("emit")
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	waitDone(t, recorder)
	if len(chunks) != 2 || string(chunks[0]) != string([]byte{4}) || string(chunks[1]) != string([]byte{1, 2, 3}) {
		t.Fatalf("chunks = %v", chunks)
	}
	recorder.Dispose()
}

func TestBrowserRecorder_ConversionFailureClosesDone(t *testing.T) {
	cleanup := installRecorderFake()
	defer cleanup()
	stream := js.Global().Get("Object").New()
	recorder, err := NewBrowserRecorder(stream, "audio/webm", func([]byte) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	recorder.recorder.Call("emitReject")
	if err := recorder.Stop(); err != nil {
		t.Fatal(err)
	}
	waitDone(t, recorder)
	if recorder.Err() == nil || !strings.Contains(recorder.Err().Error(), "conversion") {
		t.Fatalf("err = %v", recorder.Err())
	}
	recorder.Dispose()
}

func TestBrowserRecorder_QueueOverflowClosesDone(t *testing.T) {
	cleanup := installRecorderFake()
	defer cleanup()
	stream := js.Global().Get("Object").New()
	release := make(chan struct{})
	recorder, err := NewBrowserRecorder(stream, "audio/webm", func([]byte) bool {
		<-release
		return true
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < defaultPTTQueueSize+2; i++ {
		recorder.recorder.Call("emit")
	}
	close(release)
	waitDone(t, recorder)
	if recorder.Err() == nil || !strings.Contains(recorder.Err().Error(), "queue is full") {
		t.Fatal("overflow error missing")
	}
	recorder.Dispose()
}

func TestBrowserRecorder_StartStopErrorsAreReturned(t *testing.T) {
	cleanup := installRecorderFake()
	defer cleanup()
	stream := js.Global().Get("Object").New()
	recorder, err := NewBrowserRecorder(stream, "audio/webm", func([]byte) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	defer recorder.Dispose()
	js.Global().Set("DFThrowStart", true)
	if err := recorder.Start(); err == nil {
		t.Fatal("Start accepted a throwing MediaRecorder")
	}
	js.Global().Set("DFThrowStart", false)
	js.Global().Set("DFThrowStop", true)
	if err := recorder.Stop(); err == nil {
		t.Fatal("Stop accepted a throwing MediaRecorder")
	}
	js.Global().Set("DFThrowStop", false)
}
