package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestNudgeLines_AreNameFreeAndUseDistinctVoices(t *testing.T) {
	lines := NudgeLines()
	if len(lines) != 2 {
		t.Fatalf("got %d nudge lines", len(lines))
	}
	if lines[0].Voice == lines[1].Voice || lines[0].Text == "" || lines[1].Text == "" {
		t.Fatalf("unexpected nudges: %#v", lines)
	}
	for _, line := range lines {
		if line.ID == "" || line.Text == "Mother Vell" {
			t.Fatalf("nudge is not name-free: %#v", line)
		}
	}
}

func TestRenderNudgeLine_UsesCannedRendererAndAddsMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/text-to-speech/dm/stream" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte("nudge"))
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(filepath.Join(root, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	line := NudgeLines()[0]
	if err := RenderNudgeLine(context.Background(), server.Client(), server.URL, root, writer, line, 1); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets[line.ID]
	if asset.Kind != "AUDIO" || asset.Metadata["name_free"] != "true" || asset.Metadata["voice"] != "dm" {
		t.Fatalf("unexpected nudge asset: %#v", asset)
	}
}
