//go:build !js || !wasm

package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	spikev1 "github.com/monstercameron/DungeonFlux/scripts/spike/gen"
	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	"google.golang.org/grpc"
)

func main() {
	live := flag.Bool("live", false, "enable the paid ElevenLabs Scribe path")
	fakeText := flag.String("fake-transcript", "fake transcript", "transcript returned without -live")
	port := flag.String("port", "18120", "HTTP listen port")
	flag.Parse()
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	var stt transcriber = fakeTranscriber{text: *fakeText}
	if *live {
		key := os.Getenv("DF_ELEVENLABS_API_KEY")
		if key == "" {
			logger.Error("-live requires DF_ELEVENLABS_API_KEY")
			return
		}
		stt = newScribeTranscriber(key)
	}
	grpcServer := grpc.NewServer()
	spikev1.RegisterVoiceServer(grpcServer, voiceServer{transcriber: stt, hub: newAudioHub()})
	mux := http.NewServeMux()
	mux.Handle("/grpc", grpctunnel.Wrap(grpcServer))
	mux.HandleFunc("/phone", phonePage)
	mux.HandleFunc("/dm", dmPage)
	mux.HandleFunc("/dungeonflux.wasm", wasmFile)
	mux.HandleFunc("/wasm_exec.js", wasmExecFile)
	logger.Info("audio tunnel spike listening", "addr", ":"+*port, "live_stt", *live)
	if err := http.ListenAndServe(":"+*port, mux); err != nil {
		logger.Error("audio tunnel spike stopped", "err", err)
	}
}

func phonePage(w http.ResponseWriter, _ *http.Request) { serveHTML(w, phoneHTML) }
func dmPage(w http.ResponseWriter, _ *http.Request)    { serveHTML(w, dmHTML) }

func serveHTML(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte("<!doctype html><meta name=viewport content='width=device-width,initial-scale=1'>" + body))
}

func wasmFile(w http.ResponseWriter, _ *http.Request) {
	serveArtifact(w, filepath.Join("artifacts", "spike", "dungeonflux.wasm"), "application/wasm")
}
func wasmExecFile(w http.ResponseWriter, _ *http.Request) {
	serveArtifact(w, filepath.Join("artifacts", "spike", "wasm_exec.js"), "text/javascript")
}

func serveArtifact(w http.ResponseWriter, path, contentType string) {
	data, err := os.ReadFile(path)
	if err != nil {
		http.Error(w, "build the spike wasm first", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(data)
}

const wasmBootstrap = `<script src="/wasm_exec.js"></script><script>(async()=>{const go=new Go();const r=await WebAssembly.instantiateStreaming(fetch('/dungeonflux.wasm'),go.importObject);go.run(r.instance)})().catch(e=>document.body.append(e))</script>`
const phoneHTML = `<main><h1>Phone talk spike</h1><p id="status">Tap Join, then hold Talk.</p><button id="join">Join microphone</button><button id="talk" disabled>Hold to talk</button><pre id="transcript"></pre></main>` + wasmBootstrap
const dmHTML = `<main><h1>DM audio spike</h1><p id="status">Waiting for phone audio…</p><audio id="audio" controls></audio></main>` + wasmBootstrap
