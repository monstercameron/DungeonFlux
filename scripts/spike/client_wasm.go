//go:build js && wasm

package main

import (
	"context"
	"fmt"
	"syscall/js"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
	"github.com/monstercameron/GoGRPCBridge/pkg/wasm/dialer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx := context.Background()
	conn, err := grpc.NewClient("spike", dialer.New(websocketURL()), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		show("status", err.Error())
		return
	}
	defer conn.Close()
	if js.Global().Get("document").Call("getElementById", "join").Truthy() {
		installPhone(ctx, spikev1.NewVoiceClient(conn))
	} else {
		installDM(ctx, spikev1.NewVoiceClient(conn))
	}
	select {}
}

func websocketURL() string {
	location := js.Global().Get("location")
	scheme := "ws:"
	if location.Get("protocol").String() == "https:" {
		scheme = "wss:"
	}
	return scheme + "//" + location.Get("host").String() + "/grpc"
}

func installPhone(ctx context.Context, client spikev1.VoiceClient) {
	join := js.Global().Get("document").Call("getElementById", "join")
	talk := js.Global().Get("document").Call("getElementById", "talk")
	var recorder js.Value
	var stream spikev1.Voice_TalkClient
	var chunks = make(chan []byte, 32)
	var done chan struct{}
	join.Call("addEventListener", "click", js.FuncOf(func(js.Value, []js.Value) interface{} {
		promise := js.Global().Get("navigator").Get("mediaDevices").Call("getUserMedia", map[string]interface{}{"audio": true})
		promise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			mediaStream := args[0]
			recorder = js.Global().Get("MediaRecorder").New(mediaStream)
			recorder.Set("ondataavailable", js.FuncOf(func(_ js.Value, args []js.Value) interface{} { queueBlob(args[0], chunks); return nil }))
			recorder.Set("onstop", js.FuncOf(func(js.Value, []js.Value) interface{} { close(done); return nil }))
			talk.Set("disabled", false)
			show("status", "Ready. Hold Talk.")
			return nil
		})).Call("catch", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
			show("status", args[0].Get("message").String())
			return nil
		}))
		return nil
	}))
	talk.Call("addEventListener", "pointerdown", js.FuncOf(func(js.Value, []js.Value) interface{} {
		stream, _ = client.Talk(ctx)
		done = make(chan struct{})
		go sendChunks(stream, chunks, done, recorder.Get("mimeType").String())
		go receiveTranscript(stream)
		recorder.Call("start", 100)
		show("status", "Recording…")
		return nil
	}))
	talk.Call("addEventListener", "pointerup", js.FuncOf(func(js.Value, []js.Value) interface{} {
		recorder.Call("stop")
		show("status", "Transcribing…")
		return nil
	}))
}

func queueBlob(blob js.Value, chunks chan<- []byte) {
	promise := blob.Call("arrayBuffer")
	promise.Call("then", js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		array := js.Global().Get("Uint8Array").New(args[0])
		bytes := make([]byte, array.Length())
		js.CopyBytesToGo(bytes, array)
		chunks <- bytes
		return nil
	}))
}

func sendChunks(stream spikev1.Voice_TalkClient, chunks <-chan []byte, done <-chan struct{}, mimeType string) {
	if stream == nil {
		return
	}
	for {
		select {
		case chunk := <-chunks:
			_ = stream.Send(&spikev1.AudioChunk{PcmS16Le: chunk, MimeType: mimeType})
		case <-done:
			_ = stream.Send(&spikev1.AudioChunk{Final: true})
			_ = stream.CloseSend()
			return
		}
	}
}

func receiveTranscript(stream spikev1.Voice_TalkClient) {
	if stream == nil {
		return
	}
	result, err := stream.CloseAndRecv()
	if err != nil {
		show("status", err.Error())
		return
	}
	show("transcript", result.GetText())
}

func installDM(ctx context.Context, client spikev1.VoiceClient) {
	stream, err := client.Listen(ctx, &spikev1.ListenRequest{SeatToken: "spike-dm"})
	if err != nil {
		show("status", err.Error())
		return
	}
	for {
		frame, err := stream.Recv()
		if err != nil {
			show("status", err.Error())
			return
		}
		playPCM(frame.GetPcmS16Le(), frame.GetSampleRate())
	}
}

func playPCM(data []byte, rate uint32) {
	if rate == 0 {
		rate = 24000
	}
	context := js.Global().Get("AudioContext").New()
	buffer := context.Call("createBuffer", 1, len(data)/2, rate)
	channel := buffer.Call("getChannelData", 0)
	for i := 0; i+1 < len(data); i += 2 {
		sample := int16(data[i]) | int16(data[i+1])<<8
		channel.SetIndex(i/2, float64(sample)/32768)
	}
	source := context.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Call("connect", context.Get("destination"))
	source.Call("start")
}

func show(id, text string) {
	element := js.Global().Get("document").Call("getElementById", id)
	if element.Truthy() {
		element.Set("textContent", text)
	}
}

var _ = fmt.Sprintf
