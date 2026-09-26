package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestRunThrallLoopsJob_RegistersSourcesAndMeasuredLoops(t *testing.T) {
	root := t.TempDir()
	still := filepath.Join(root, "still.png")
	cutout := filepath.Join(root, "cutout.png")
	if err := os.WriteFile(still, []byte("still"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cutout, []byte("cutout"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request loopRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Width != 480 || request.Height != 854 || request.NoAudio != true {
			t.Fatalf("unexpected request: %#v", request)
		}
		json.NewEncoder(w).Encode(loopResponse{VideoURL: server.URL + "/loop.mp4"})
	}))
	defer server.Close()
	server.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/loop.mp4" {
			w.Write([]byte("video"))
			return
		}
		var request loopRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		json.NewEncoder(w).Encode(loopResponse{VideoURL: server.URL + "/loop.mp4"})
	})
	options := LoopOptions{Endpoint: server.URL, APIKey: "test", Specs: []LoopSpec{{LogicalName: "thrall_attack", Prompt: "slam", Take: 1, DurationMS: 4000, ContactMS: 1375}}}
	if err := RunThrallLoopsJob(context.Background(), writer, options, still, cutout); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets["thrall_attack"]
	if asset.Kind != "VIDEO_LOOP" || asset.DurationMS != 4000 || asset.ContactMS != 1375 {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if _, err := os.Stat(filepath.Join(root, asset.Takes[0].Path)); err != nil {
		t.Fatal(err)
	}
}

func TestLoopJob_DryRunAndValidation(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunLoopJob(context.Background(), writer, LoopOptions{DryRun: true, Specs: []LoopSpec{{LogicalName: "idle", Prompt: "breathe", Take: 1, DurationMS: 4000}}}, "VIDEO_LOOP"); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run wrote assets")
	}
	for _, spec := range []LoopSpec{{LogicalName: "../bad", Prompt: "x", Take: 1, DurationMS: 1}, {LogicalName: "bad", Take: 1, DurationMS: 1}, {LogicalName: "bad", Prompt: "x", Take: 0, DurationMS: 1}, {LogicalName: "bad", Prompt: "x", Take: 1, DurationMS: 0}} {
		if err := RunLoopJob(context.Background(), writer, LoopOptions{DryRun: true, Specs: []LoopSpec{spec}}, "VIDEO_LOOP"); err == nil {
			t.Fatalf("accepted invalid spec %#v", spec)
		}
	}
}

func TestLoopClient_RejectsErrorsAndEmptyVideo(t *testing.T) {
	for _, response := range []loopResponse{{Error: "nope"}, {VideoURL: ""}} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if response.Error != "" {
				w.WriteHeader(http.StatusBadRequest)
			}
			json.NewEncoder(w).Encode(response)
		}))
		_, err := (LoopClient{Endpoint: server.URL, APIKey: "key"}).Generate(context.Background(), LoopSpec{Prompt: "x"})
		server.Close()
		if err == nil {
			t.Fatal("accepted invalid loop response")
		}
	}
	if _, err := (LoopClient{}).Generate(context.Background(), LoopSpec{Prompt: "x"}); err == nil {
		t.Fatal("accepted missing credentials")
	}
}
