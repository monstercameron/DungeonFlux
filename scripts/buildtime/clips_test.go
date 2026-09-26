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

func TestBuildVideoRequest_ClipsArePinnedToSpec(t *testing.T) {
	for _, spec := range ClipSpecs() {
		data, err := BuildVideoRequest(spec)
		if err != nil {
			t.Fatal(err)
		}
		var req VideoRequest
		if err := json.Unmarshal(data, &req); err != nil {
			t.Fatal(err)
		}
		if req.Duration != 5 || req.Resolution != "720p" || req.GenerateAudio || req.Prompt == "" {
			t.Fatalf("unexpected request for %s: %#v", spec.ID, req)
		}
	}
}

func TestBuildVideoRequest_RejectsInvalidSpecs(t *testing.T) {
	cases := []ClipSpec{
		{Prompt: "x", FirstFrameURL: "x", DurationMS: 5000, Resolution: "720p", Take: 1},
		{ID: "x", FirstFrameURL: "x", DurationMS: 5000, Resolution: "720p", Take: 1},
		{ID: "x", Prompt: "x", DurationMS: 5000, Resolution: "720p", Take: 1},
		{ID: "x", Prompt: "x", FirstFrameURL: "x", DurationMS: 500, Resolution: "720p", Take: 1},
		{ID: "x", Prompt: "x", FirstFrameURL: "x", DurationMS: 5000, Take: 1},
		{ID: "x", Prompt: "x", FirstFrameURL: "x", DurationMS: 5000, Resolution: "720p"},
		{ID: "x", Prompt: "x", FirstFrameURL: "x", LastFrameURL: "y", DurationMS: 5000, Resolution: "720p", Take: 1},
	}
	for i, spec := range cases {
		if _, err := BuildVideoRequest(spec); err == nil {
			t.Fatalf("case %d accepted invalid spec", i)
		}
	}
}

func TestClipJob_DryRunLeavesManifestEmpty(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := ClipJob(VideoClient{}, t.TempDir(), true).Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatal("dry-run registered assets")
	}
}

func TestVideoClient_RenderVideo_DownloadsAndRecordsManifest(t *testing.T) {
	root := t.TempDir()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			json.NewEncoder(w).Encode(videoSubmitResponse{VideoURL: "http://" + r.Host + "/clip.mp4", Status: "done"})
			return
		}
		if r.URL.Path == "/clip.mp4" {
			w.Write([]byte("fake mp4"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	writer, err := NewManifestWriter(filepath.Join(root, "buildtime"))
	if err != nil {
		t.Fatal(err)
	}
	spec := ClipSpecs()[0]
	client := VideoClient{HTTPClient: server.Client(), Endpoint: server.URL, APIKey: "secret"}
	if err := client.RenderVideo(context.Background(), writer, root, spec); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets[spec.ID]
	if asset.Kind != "VIDEO" || asset.DurationMS != 5000 || asset.Metadata["audio"] != "false" || len(asset.Takes) != 1 {
		t.Fatalf("unexpected manifest asset: %#v", asset)
	}
	if _, err := os.Stat(filepath.Join(writer.root, asset.Takes[0].Path)); err != nil {
		t.Fatal(err)
	}
}

func TestVideoClient_RenderVideo_RejectsUnfinishedJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(videoSubmitResponse{RequestID: "queued", Status: "queued"})
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	client := VideoClient{HTTPClient: server.Client(), Endpoint: server.URL}
	if err := client.RenderVideo(context.Background(), writer, t.TempDir(), ClipSpecs()[0]); err == nil {
		t.Fatal("accepted unfinished job")
	}
}
