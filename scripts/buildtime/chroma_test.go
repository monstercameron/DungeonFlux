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

func TestRunChromaLatencyJob_SavesMeasuredSamples(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.Write([]byte("sample"))
			return
		}
		var request loopRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Width != 480 || request.Height != 854 || request.DurationSeconds != 4 {
			t.Fatalf("unexpected request: %#v", request)
		}
		json.NewEncoder(w).Encode(loopResponse{VideoURL: server.URL + "/sample.mp4"})
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunChromaLatencyJob(context.Background(), writer, ChromaOptions{Endpoint: server.URL, APIKey: "test", Samples: 3}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		asset := writer.manifest.Assets["chroma_latency_"+string(rune('0'+i))]
		if asset.Kind != "CHROMA_SAMPLE" || asset.DurationMS != 4000 || asset.Metadata["elapsed_ms"] == "" {
			t.Fatalf("unexpected sample: %#v", asset)
		}
		if _, err := os.Stat(filepath.Join(root, asset.Takes[0].Path)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRunChromaLatencyJob_DryRunAndValidation(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunChromaLatencyJob(context.Background(), writer, ChromaOptions{DryRun: true, Samples: 1}); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run wrote samples")
	}
	if err := RunChromaLatencyJob(context.Background(), writer, ChromaOptions{Samples: -1}); err == nil {
		t.Fatal("accepted invalid sample count")
	}
}
