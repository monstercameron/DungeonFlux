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

func TestMusicTracks_HaveCompleteCatalogue(t *testing.T) {
	tracks := MusicTracks()
	if len(tracks) != 12 {
		t.Fatalf("got %d music tracks", len(tracks))
	}
	seen := make(map[string]bool)
	for _, track := range tracks {
		if track.ID == "" || track.BPM <= 0 || track.DurationMS < 3000 || track.Prompt == "" || seen[track.ID] {
			t.Fatalf("invalid or duplicate track: %#v", track)
		}
		seen[track.ID] = true
	}
	for _, id := range []string{"THEME_MAIN", "COMBAT_SKIRMISH_LOOP", "END_CARD_THEME"} {
		if !seen[id] {
			t.Fatalf("missing track %q", id)
		}
	}
}

func TestBuildMusicRequest_UsesPinnedModelAndSeed(t *testing.T) {
	data, err := BuildMusicRequest(MusicTracks()[0])
	if err != nil {
		t.Fatal(err)
	}
	var request MusicRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.ModelID != "music_v2_5" || request.Seed != MusicSeed("THEME_MAIN") || !request.StoreForInpaint {
		t.Fatalf("unexpected request: %#v", request)
	}
	if len(request.CompositionPlan.Chunks) != 1 {
		t.Fatalf("missing composition chunk: %#v", request)
	}
	if _, err := BuildMusicRequest(MusicTrack{ID: "bad", DurationMS: 1000, BPM: 80, Prompt: "x"}); err == nil {
		t.Fatal("accepted short track")
	}
}

func TestRenderMusic_StoresTrackMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/music/detailed" || r.URL.Query().Get("output_format") != "mp3_44100_192" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("music"))
	}))
	defer server.Close()
	root := t.TempDir()
	manifestRoot := filepath.Join(root, "manifest")
	writer, err := NewManifestWriter(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	track := MusicTracks()[0]
	if err := RenderMusic(context.Background(), server.Client(), server.URL, root, writer, track, 1); err != nil {
		t.Fatal(err)
	}
	stored := writer.manifest.Assets[track.ID]
	if stored.Metadata["bpm"] != "80" || stored.Metadata["loop_end_ms"] != "96000" || stored.DurationMS != 96000 {
		t.Fatalf("unexpected metadata: %#v", stored)
	}
	data, err := os.ReadFile(filepath.Join(manifestRoot, stored.Takes[0].Path))
	if err != nil || string(data) != "music" {
		t.Fatalf("stored music mismatch: %v %q", err, data)
	}
}
