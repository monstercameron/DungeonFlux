//go:build live

package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLiveMusicJob(t *testing.T) {
	if os.Getenv("DF_LIVE") != "1" {
		t.Skip("DF_LIVE=1 is required")
	}
	options := DefaultMusicOptions()
	if err := PrintMusicPlan(os.Stdout, options); err != nil {
		t.Fatal(err)
	}
	endpoint := os.Getenv("DF_ELEVENLABS_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.elevenlabs.io/v1"
	}
	root := "artifacts/runtime/buildtime"
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 20 * time.Minute}
	job := MusicJobWithOptions(client, endpoint, root, 1, options)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "music", "costs.jsonl")); err != nil {
		t.Fatal(err)
	}
}
