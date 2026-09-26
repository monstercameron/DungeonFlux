package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAmbienceScenes_DerivePromptsFromOneShotBeats(t *testing.T) {
	scenes := AmbienceScenes()
	if len(scenes) != 5 {
		t.Fatalf("got %d scenes, want 5", len(scenes))
	}
	ids := map[string]bool{}
	for _, scene := range scenes {
		if ids[scene.ID] || scene.ID == "" || scene.BeatID == "" || scene.Prompt == "" {
			t.Fatalf("invalid scene: %#v", scene)
		}
		ids[scene.ID] = true
		if scene.DurationSeconds != 30 || scene.Crossfade != 3*time.Second || scene.TargetLUFS != -24 {
			t.Fatalf("unexpected timing or level: %#v", scene)
		}
		if !strings.Contains(scene.Prompt, "Story beat:") || !strings.Contains(scene.Prompt, "no music") {
			t.Fatalf("prompt is not scene-derived: %q", scene.Prompt)
		}
	}
	for _, id := range []string{"ambience_tavern_rain", "ambience_harbor_night", "ambience_bell_tower_wind", "ambience_combat_tension", "ambience_dawn"} {
		if !ids[id] {
			t.Fatalf("missing ambience %q", id)
		}
	}
}

func TestBuildAmbienceRequest_UsesDurationAndPrompt(t *testing.T) {
	scene := AmbienceScenes()[0]
	data, err := BuildAmbienceRequest(scene)
	if err != nil {
		t.Fatal(err)
	}
	var request AmbienceRequest
	if err := json.Unmarshal(data, &request); err != nil {
		t.Fatal(err)
	}
	if request.DurationSeconds != 30 || request.Text != scene.Prompt || request.PromptInfluence != 0.3 {
		t.Fatalf("unexpected request: %#v", request)
	}
	for _, invalid := range []AmbienceScene{
		{ID: "missing-prompt", BeatID: "opening", DurationSeconds: 30, Crossfade: 3 * time.Second},
		{ID: "short", BeatID: "opening", Prompt: "x", DurationSeconds: 29, Crossfade: 3 * time.Second},
		{ID: "long", BeatID: "opening", Prompt: "x", DurationSeconds: 61, Crossfade: 3 * time.Second},
		{ID: "seam", BeatID: "opening", Prompt: "x", DurationSeconds: 30, Crossfade: 15 * time.Second},
	} {
		if _, err := BuildAmbienceRequest(invalid); err == nil {
			t.Fatalf("accepted invalid scene: %#v", invalid)
		}
	}
}

func TestLoopPlan_RejectsInvalidTimingAndPreservesDuration(t *testing.T) {
	plan, err := LoopPlan(AmbienceScene{DurationSeconds: 45, Crossfade: 2500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	if plan.DurationSeconds != 45 || plan.CrossfadeSeconds != 2.5 {
		t.Fatalf("unexpected loop plan: %#v", plan)
	}
	filter := CrossfadeFilter(plan)
	for _, part := range []string{"atrim=start=0:end=42.500", "acrossfade=d=2.500", "concat=n=2", "loudnorm=I=-24"} {
		if !strings.Contains(filter, part) {
			t.Fatalf("filter %q missing %q", filter, part)
		}
	}
	for _, scene := range []AmbienceScene{
		{DurationSeconds: 29, Crossfade: time.Second},
		{DurationSeconds: 61, Crossfade: time.Second},
		{DurationSeconds: 30, Crossfade: 0},
		{DurationSeconds: 30, Crossfade: 15 * time.Second},
	} {
		if _, err := LoopPlan(scene); err == nil {
			t.Fatalf("accepted invalid timing: %#v", scene)
		}
	}
}

func TestAmbienceJob_DryRunDoesNotWriteAssets(t *testing.T) {
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	job := AmbienceJob(nil, "", "", filepath.Join(root, "ambience"), 1, true)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != 0 {
		t.Fatalf("dry-run wrote assets: %#v", writer.manifest.Assets)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("dry-run left manifest lock: %v", err)
	}
}

func TestRenderAmbience_PostProcessesAndRegistersLoop(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	root := t.TempDir()
	data := makeTestMP3(t, root)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sound-generation" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	manifestRoot := filepath.Join(root, "manifest")
	writer, err := NewManifestWriter(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	scene := AmbienceScenes()[0]
	outputDir := filepath.Join(root, "ambience")
	if err := RenderAmbience(context.Background(), server.Client(), server.URL, "ffmpeg", outputDir, writer, scene, 1); err != nil {
		t.Fatal(err)
	}
	asset := writer.manifest.Assets[scene.ID]
	if asset.Kind != "AMBIENCE" || asset.DurationMS != 30000 || asset.ContactMS != 0 || asset.Metadata["target_lufs"] != "-24" {
		t.Fatalf("unexpected asset metadata: %#v", asset)
	}
	if _, err := os.Stat(filepath.Join(outputDir, scene.ID+".ogg")); err != nil {
		t.Fatal(err)
	}
}

func TestAmbienceJob_LiveTakesManifestLockAndWritesAllScenes(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg is not installed")
	}
	data := makeTestMP3(t, t.TempDir())
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/sound-generation" {
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
		_, _ = w.Write(data)
	}))
	defer server.Close()
	root := t.TempDir()
	manifestRoot := filepath.Join(root, "manifest")
	writer, err := NewManifestWriter(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	outputDir := filepath.Join(root, "ambience")
	job := AmbienceJob(server.Client(), server.URL, "ffmpeg", outputDir, 1, false)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if requests != len(AmbienceScenes()) || len(writer.manifest.Assets) != len(AmbienceScenes()) {
		t.Fatalf("requests=%d assets=%d", requests, len(writer.manifest.Assets))
	}
	if _, err := os.Stat(filepath.Join(manifestRoot, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRenderAmbience_RejectsTransportAndOutputFailures(t *testing.T) {
	scene := AmbienceScenes()[0]
	root := t.TempDir()
	writer, err := NewManifestWriter(filepath.Join(root, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderAmbience(context.Background(), nil, "http://example.test", "ffmpeg", root, writer, scene, 1); err == nil {
		t.Fatal("nil client accepted")
	}
	if err := RenderAmbience(context.Background(), &http.Client{}, "http://example.test", "ffmpeg", root, nil, scene, 1); err == nil {
		t.Fatal("nil writer accepted")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	if err := RenderAmbience(context.Background(), server.Client(), server.URL, "ffmpeg", root, writer, scene, 1); err == nil {
		t.Fatal("bad response accepted")
	}
	server.Close()
	if err := RenderAmbience(context.Background(), &http.Client{}, "://bad", "ffmpeg", root, writer, scene, 1); err == nil {
		t.Fatal("bad endpoint accepted")
	}
	outputFile := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(outputFile, []byte("file"), 0o644); err != nil {
		t.Fatal(err)
	}
	dataServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("audio"))
	}))
	defer dataServer.Close()
	if err := RenderAmbience(context.Background(), dataServer.Client(), dataServer.URL, "ffmpeg", outputFile, writer, scene, 1); err == nil {
		t.Fatal("file output path accepted")
	}
	if err := RenderAmbience(context.Background(), dataServer.Client(), dataServer.URL, "missing-ffmpeg", root, writer, scene, 1); err == nil {
		t.Fatal("missing ffmpeg accepted")
	}
}

func TestRunAmbience_RequiresKeyForLiveMode(t *testing.T) {
	t.Setenv("DF_ELEVENLABS_API_KEY", "")
	if err := runAmbience([]string{"-root", t.TempDir()}); err == nil {
		t.Fatal("live mode accepted missing API key")
	}
	if err := runAmbience([]string{"-root", t.TempDir(), "-dry-run"}); err != nil {
		t.Fatal(err)
	}
}

func TestBuildtimePlans_AreOfflineAndBounded(t *testing.T) {
	sfxPlan, err := PlanSFX(SFXAssets(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if sfxPlan.Requests != len(SFXAssets()) || sfxPlan.EstimatedSeconds <= 0 || sfxPlan.EstimatedCostUSD <= 0 {
		t.Fatalf("unexpected SFX plan: %#v", sfxPlan)
	}
	if _, err := PlanSFX(SFXAssets(), 4); err == nil {
		t.Fatal("accepted too many SFX takes")
	}
	musicPlan, err := MusicDryRun(MusicOptions{Takes: 1, MaxConcurrent: 1})
	if err != nil {
		t.Fatal(err)
	}
	if musicPlan.Requests != len(MusicTracks()) || musicPlan.GeneratedSeconds <= 0 {
		t.Fatalf("unexpected music plan: %#v", musicPlan)
	}
	var output strings.Builder
	if err := PrintMusicPlan(&output, MusicOptions{Takes: 1, MaxConcurrent: 1}); err != nil || !strings.Contains(output.String(), `"requests"`) {
		t.Fatalf("music plan output: %v %q", err, output.String())
	}
	if err := PrintMusicPlan(nil, MusicOptions{}); err == nil {
		t.Fatal("nil music plan output accepted")
	}
}

func TestBuildtimeAudioHelpers_NormalizeIDsAndMetadata(t *testing.T) {
	if got := audioOutputDir(filepath.Join("root", "audio")); got != filepath.Join("root", "audio") {
		t.Fatalf("audio directory changed: %q", got)
	}
	if got := audioOutputDir("root"); got != filepath.Join("root", "audio") {
		t.Fatalf("audio directory = %q", got)
	}
	if got := resolveVoiceID("dm"); got == "" {
		t.Fatal("default voice is empty")
	}
	t.Setenv("DF_ELEVENLABS_VOICE_DM", "test-voice")
	if got := resolveVoiceID("dm"); got != "test-voice" {
		t.Fatalf("configured voice = %q", got)
	}
	plan := CannedTTSPlan()
	nudges := NudgeTTSPlan()
	if plan.Requests != len(CannedLines()) || plan.Characters <= nudges.Characters || nudges.Requests != len(NudgeLines()) {
		t.Fatalf("unexpected TTS plans: %#v %#v", plan, nudges)
	}
	metadata := audioMetadata(CannedLines()[0], renderedAudio{Characters: 3, CostUSD: 0.01, DurationMS: 25})
	if metadata["sample_rate"] != "24000" || !strings.Contains(summaryFile(renderedAudio{Path: "x.pcm", DurationMS: 25, Characters: 3}, "audio"), "duration_ms=25") {
		t.Fatalf("unexpected audio metadata: %#v", metadata)
	}
}

func TestBuildtimeJobWrappersAndSummaries_AreDeterministic(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RunJobs(context.Background(), writer, nil); err != nil {
		t.Fatal(err)
	}
	musicJob := MusicJobWithOptions(nil, "", "", 1, MusicOptions{DryRun: true, Takes: 1, MaxConcurrent: 1})
	if err := musicJob.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if err := MusicJobWithOptions(nil, "", "", 1, MusicOptions{Takes: 1, MaxConcurrent: musicConcurrency + 1}).Run(context.Background(), writer); err == nil {
		t.Fatal("music job accepted excessive concurrency")
	}
	asset := SFXAssets()[0]
	good := SFXMediaStats{DurationSeconds: 2, IntegratedLUFS: -16}
	if !acceptableSFXStats(asset, good) || acceptableSFXStats(asset, SFXMediaStats{DurationSeconds: 40, IntegratedLUFS: -16}) {
		t.Fatal("unexpected SFX media acceptance")
	}
	if sfxScore(asset, good) != 0 {
		t.Fatalf("perfect SFX score = %v", sfxScore(asset, good))
	}
	var log strings.Builder
	writeSFXLog(&log, asset, 1, good)
	writeSFXSummary(&log, SFXPlan{Requests: 1}, SFXSummary{Requests: 1, Seconds: 2, Selected: 1})
	if !strings.Contains(log.String(), "sfx_summary") {
		t.Fatalf("summary log = %q", log.String())
	}
}

func TestExistingAudioJobs_RunAgainstOfflineFixture(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/sound-generation") {
			_, _ = w.Write([]byte("effect"))
			return
		}
		_, _ = w.Write(make([]byte, 4800))
	}))
	defer server.Close()
	root := t.TempDir()
	sfxWriter, err := NewManifestWriter(filepath.Join(root, "sfx-manifest"))
	if err != nil {
		t.Fatal(err)
	}
	if err := SFXJob(server.Client(), server.URL, filepath.Join(root, "sfx"), 1).Run(context.Background(), sfxWriter); err != nil {
		t.Fatal(err)
	}
	if len(sfxWriter.manifest.Assets) != len(SFXAssets()) {
		t.Fatalf("SFX assets = %d", len(sfxWriter.manifest.Assets))
	}
	cannedWriter, err := NewManifestWriter(filepath.Join(root, "canned-manifest"))
	if err != nil {
		t.Fatal(err)
	}
	if err := CannedJob(server.Client(), server.URL, filepath.Join(root, "canned"), 1).Run(context.Background(), cannedWriter); err != nil {
		t.Fatal(err)
	}
	if len(cannedWriter.manifest.Assets) != len(CannedLines()) {
		t.Fatalf("canned assets = %d", len(cannedWriter.manifest.Assets))
	}
}

func makeTestMP3(t *testing.T, root string) []byte {
	t.Helper()
	source := filepath.Join(root, "source.mp3")
	command := exec.Command("ffmpeg", "-y", "-v", "error", "-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo", "-t", "30", "-c:a", "libmp3lame", "-b:a", "64k", source)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("make source audio: %v: %s", err, output)
	}
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestManifestLock_ReloadsExistingManifestAndCleansUp(t *testing.T) {
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.AddFile("existing", "IMAGE", writeTestAsset(t, root, "old"), 1); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.AddFile("local", "IMAGE", writeTestAsset(t, root, "local"), 1); err != nil {
		t.Fatal(err)
	}
	lock, err := LockManifest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := reloadManifest(writer); err != nil {
		_ = UnlockManifest(lock)
		t.Fatal(err)
	}
	if _, ok := writer.manifest.Assets["local"]; ok {
		t.Fatal("reload retained stale local asset")
	}
	if _, ok := writer.manifest.Assets["existing"]; !ok {
		t.Fatal("reload lost existing asset")
	}
	if err := UnlockManifest(lock); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("manifest lock remains: %v", err)
	}
}

func writeTestAsset(t *testing.T, root, contents string) string {
	t.Helper()
	path := filepath.Join(root, contents+".bin")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
