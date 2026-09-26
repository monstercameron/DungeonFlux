package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
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
	if len(request.CompositionPlan.Chunks) != 4 {
		t.Fatalf("missing composition chunk: %#v", request)
	}
	if _, err := BuildMusicRequest(MusicTrack{ID: "bad", DurationMS: 1000, BPM: 80, Prompt: "x"}); err == nil {
		t.Fatal("accepted short track")
	}
}

func TestBuildMusicRequestWithTheme_UsesConditioningAndOmitsEndCardSeed(t *testing.T) {
	data, err := BuildMusicRequestWithTheme(MusicTracks()[1], "theme-song")
	if err != nil {
		t.Fatal(err)
	}
	var request MusicRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.ConditioningRef == nil || request.ConditioningRef.SongID != "theme-song" || request.ConditioningRef.Range.EndMS != 24000 {
		t.Fatalf("missing theme conditioning: %#v", request.ConditioningRef)
	}
	if request.Seed == 0 || request.ConditionStrength != "medium" {
		t.Fatalf("unexpected conditioned request: %#v", request)
	}
	data, err = BuildMusicRequestWithTheme(MusicTracks()[11], "theme-song")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "" || string(data) == "null" {
		t.Fatal("empty end-card request")
	}
	var endCard MusicRequest
	if err := json.Unmarshal(data, &endCard); err != nil {
		t.Fatal(err)
	}
	if endCard.Seed != 0 || endCard.ConditionStrength != "high" {
		t.Fatalf("unexpected end-card request: %#v", endCard)
	}
}

func TestMusicDryRun_ReportsThreeTakesAndBudget(t *testing.T) {
	options := DefaultMusicOptions()
	plan, err := MusicDryRun(options)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Tracks != 12 || plan.Takes != 3 || plan.Requests != 36 || plan.MaxConcurrent != 2 || plan.EstimatedCostUSD <= 0 {
		t.Fatalf("unexpected plan: %+v", plan)
	}
	var output bytes.Buffer
	if err := PrintMusicPlan(&output, options); err != nil || output.Len() == 0 {
		t.Fatalf("dry-run output: %v %q", err, output.String())
	}
}

func TestMusicDryRun_SelectedTracks(t *testing.T) {
	plan, err := MusicDryRun(MusicOptions{Takes: 1, MaxConcurrent: 1, TrackIDs: []string{"CLIFF_TENSION_BED", "END_CARD_THEME"}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Tracks != 2 || plan.Requests != 2 || plan.GeneratedSeconds != 48 {
		t.Fatalf("unexpected selected plan: %+v", plan)
	}
	if _, err := MusicDryRun(MusicOptions{TrackIDs: []string{"missing"}}); err == nil {
		t.Fatal("unknown track accepted")
	}
}

func TestFoldMeasuredTempo_FoldsTempoFamily(t *testing.T) {
	for _, tc := range []struct {
		name     string
		measured float64
		want     float64
	}{
		{name: "target", measured: 80, want: 80},
		{name: "double", measured: 160, want: 80},
		{name: "half", measured: 40, want: 80},
		{name: "within tolerance", measured: 82.2, want: 82.2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := foldMeasuredTempo(tc.measured, 80)
			if err != nil {
				t.Fatalf("foldMeasuredTempo: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %.2f, want %.2f", got, tc.want)
			}
		})
	}
	for _, tc := range []struct {
		name     string
		measured float64
		expected float64
	}{
		{name: "too fast", measured: 90, expected: 80},
		{name: "zero measured", measured: 0, expected: 80},
		{name: "zero target", measured: 80, expected: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := foldMeasuredTempo(tc.measured, tc.expected); err == nil {
				t.Fatal("foldMeasuredTempo accepted invalid tempo")
			}
		})
	}
}

func TestProcessMusicAudio_UsesOneFilterGraph(t *testing.T) {
	for _, tc := range []struct {
		name string
		loop bool
	}{
		{name: "loop", loop: true},
		{name: "one-shot", loop: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			capture := filepath.Join(dir, "args.txt")
			ffmpeg := filepath.Join(dir, "ffmpeg.cmd")
			script := fmt.Sprintf("@echo %%* > \"%s\"\n", capture)
			if err := os.WriteFile(ffmpeg, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			track := MusicTrack{BPM: 80, DurationMS: 96000, LoudnessLUFS: -20, Loop: tc.loop}
			if err := processMusicAudio(context.Background(), ffmpeg, "input.mp3", "output.opus", track, beatMeasurement{DownbeatMS: 125}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			args := string(data)
			if !strings.Contains(args, "-filter_complex") || !strings.Contains(args, "[normalized]") || strings.Contains(args, "-af") {
				t.Fatalf("unexpected ffmpeg args: %q", args)
			}
			if tc.loop != strings.Contains(args, "acrossfade") {
				t.Fatalf("loop filter mismatch: %q", args)
			}
		})
	}
}

func TestMusicJob_RetriesOnceAndWritesCostLog(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/music/detailed" || r.URL.Query().Get("output_format") != "mp3_44100_192" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		w.Header().Set("song_id", "theme-song")
		_, _ = w.Write([]byte("music"))
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(filepath.Join(root, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	options := MusicOptions{Takes: 1, MaxConcurrent: 2, ProcessAudio: false}
	job := MusicJobWithOptions(server.Client(), server.URL, filepath.Join(root, "output"), 1, options)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != int32(len(MusicTracks())+1) {
		t.Fatalf("expected one retry, got %d requests", requests.Load())
	}
	if _, err := os.Stat(filepath.Join(writer.root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("manifest lock remains: %v", err)
	}
	costs, err := os.ReadFile(filepath.Join(root, "output", "music", "costs.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := bytes.Count(costs, []byte{'\n'}); lines != len(MusicTracks())+1 {
		t.Fatalf("expected %d cost records, got %d", len(MusicTracks())+1, lines)
	}
	if got := writer.manifest.Assets["TAVERN_WARM_LOOP"].Metadata["model"]; got != musicModel {
		t.Fatalf("missing model metadata: %q", got)
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
