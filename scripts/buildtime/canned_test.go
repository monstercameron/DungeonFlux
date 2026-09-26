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

func TestCannedLines_HaveStableContent(t *testing.T) {
	lines := CannedLines()
	if len(lines) != 11 {
		t.Fatalf("got %d canned lines", len(lines))
	}
	seen := make(map[string]bool)
	for _, line := range lines {
		if line.ID == "" || line.Voice == "" || line.Text == "" || seen[line.ID] {
			t.Fatalf("invalid or duplicate line: %#v", line)
		}
		seen[line.ID] = true
	}
}

func TestBuildCannedTTSRequest_EncodesFlashModel(t *testing.T) {
	data, err := BuildCannedTTSRequest(CannedLines()[0])
	if err != nil {
		t.Fatal(err)
	}
	var request CannedTTSRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.ModelID != "eleven_flash_v2_5" || request.Text == "" {
		t.Fatalf("unexpected request: %#v", request)
	}
	if _, err := BuildCannedTTSRequest(CannedLine{}); err == nil {
		t.Fatal("accepted incomplete line")
	}
}

func TestRenderCannedLine_WritesManifestAssetAndHeader(t *testing.T) {
	var gotPath, gotVoice string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotVoice = r.URL.Path, r.URL.Query().Get("output_format")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pcm bytes"))
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(filepath.Join(root, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	line := CannedLines()[0]
	if err := RenderCannedLine(context.Background(), server.Client(), server.URL, root, writer, line, 1); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/text-to-speech/dm/stream" || gotVoice != "pcm_24000" {
		t.Fatalf("unexpected request path/format: %s %s", gotPath, gotVoice)
	}
	asset := writer.manifest.Assets[line.ID]
	if asset.Kind != "AUDIO" || len(asset.Takes) != 1 {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	data, err := os.ReadFile(filepath.Join(filepath.Dir(filepath.Join(root, "manifest")), "manifest", asset.Takes[0].Path))
	if err != nil || string(data) != "pcm bytes" {
		t.Fatalf("stored audio mismatch: %v %q", err, data)
	}
}

func TestRenderCannedLine_ReportsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusBadGateway) }))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderCannedLine(context.Background(), server.Client(), server.URL, t.TempDir(), writer, CannedLines()[0], 1); err == nil {
		t.Fatal("accepted failed response")
	}
}
