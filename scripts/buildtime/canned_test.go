package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
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

func TestCannedPlans_CountCharactersAndUseAudioDirectory(t *testing.T) {
	canned := CannedTTSPlan()
	if canned.Requests != len(CannedLines()) || canned.Characters != 1413 || canned.EstimatedCostUSD <= 0 {
		t.Fatalf("canned plan = %#v", canned)
	}
	nudges := NudgeTTSPlan()
	if nudges.Requests != len(NudgeLines()) || nudges.Characters != 41 || nudges.EstimatedCostUSD <= 0 {
		t.Fatalf("nudge plan = %#v", nudges)
	}
	if got := filepath.Base(audioOutputDir(filepath.Join(t.TempDir(), "buildtime"))); got != "audio" {
		t.Fatalf("audio output base = %q", got)
	}
	if got := audioOutputDir(filepath.Join(t.TempDir(), "audio")); filepath.Base(got) != "audio" {
		t.Fatalf("audio output was nested: %q", got)
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

func TestRenderCannedLine_RejectsInvalidRequestInputs(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	line := CannedLines()[0]
	for name, client := range map[string]*http.Client{"nil client": nil, "valid client": &http.Client{}} {
		t.Run(name, func(t *testing.T) {
			if err := RenderCannedLine(context.Background(), client, "https://example.invalid", t.TempDir(), writer, line, 1); err == nil {
				t.Fatal("invalid client accepted")
			}
		})
	}
	badID := line
	badID.ID = "../audio"
	if err := RenderCannedLine(context.Background(), &http.Client{}, "https://example.invalid", t.TempDir(), writer, badID, 1); err == nil {
		t.Fatal("path traversal ID accepted")
	}
	if err := RenderCannedLine(context.Background(), &http.Client{}, "://bad", t.TempDir(), writer, line, 1); err == nil {
		t.Fatal("malformed endpoint accepted")
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
	if gotPath != "/text-to-speech/"+resolveVoiceID("dm")+"/stream" || gotVoice != "pcm_24000" {
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

func TestRenderCannedLine_RejectsEmptyAudio(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderCannedLine(context.Background(), server.Client(), server.URL, t.TempDir(), writer, CannedLines()[0], 1); err == nil {
		t.Fatal("empty audio accepted")
	}
}

func TestRenderCannedLine_ReportsCanceledRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RenderCannedLine(ctx, server.Client(), server.URL, t.TempDir(), writer, CannedLines()[0], 1); err == nil {
		t.Fatal("canceled request accepted")
	}
}

func TestNudgeJob_LiveNormalizesAudioAndLocksManifest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Query().Get("output_format") != "pcm_24000" {
			t.Fatalf("output format = %q", r.URL.Query().Get("output_format"))
		}
		_, _ = w.Write(bytes.Repeat([]byte{0, 0}, 24000))
	}))
	defer server.Close()
	root := filepath.Join(t.TempDir(), "buildtime")
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	job := NudgeJob(server.Client(), server.URL, root, 1)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != int32(len(NudgeLines())) {
		t.Fatalf("requests = %d, want %d", requests.Load(), len(NudgeLines()))
	}
	for _, line := range NudgeLines() {
		asset, ok := writer.manifest.Assets[line.ID]
		if !ok || asset.DurationMS == 0 || asset.Metadata["normalization"] == "" || asset.Metadata["name_free"] != "true" {
			t.Errorf("manifest[%q] = %#v", line.ID, asset)
		}
		if _, err := os.Stat(filepath.Join(root, "audio", line.ID+"-take-1.pcm")); err != nil {
			t.Errorf("normalized audio %q: %v", line.ID, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("manifest lock after job: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err != nil {
		t.Fatalf("manifest after job: %v", err)
	}
}

func TestCannedJob_LiveFixtureRegistersEveryLine(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(bytes.Repeat([]byte{0, 0}, 24000))
	}))
	defer server.Close()
	root := filepath.Join(t.TempDir(), "buildtime")
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := CannedJob(server.Client(), server.URL, root, 1).Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != int32(len(CannedLines())) || len(writer.manifest.Assets) != len(CannedLines()) {
		t.Fatalf("requests=%d assets=%d", requests.Load(), len(writer.manifest.Assets))
	}
	for _, line := range CannedLines() {
		asset := writer.manifest.Assets[line.ID]
		if asset.DurationMS == 0 || asset.Metadata["normalization"] == "" {
			t.Errorf("asset %q lacks normalized metadata: %#v", line.ID, asset)
		}
	}
}

func TestCannedJob_ReleasesLockOnHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no", http.StatusBadGateway)
	}))
	defer server.Close()
	root := filepath.Join(t.TempDir(), "buildtime")
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := CannedJob(server.Client(), server.URL, root, 1).Run(context.Background(), writer); err == nil {
		t.Fatal("failed canned job accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("lock after failed canned job: %v", err)
	}
}

func TestResolveVoiceID_UsesOverrideAndKnownFallback(t *testing.T) {
	t.Setenv("DF_ELEVENLABS_VOICE_DM", "override")
	if got := resolveVoiceID("dm"); got != "override" {
		t.Fatalf("override voice = %q", got)
	}
	if got := resolveVoiceID("courier"); got == "courier" {
		t.Fatal("known voice did not resolve")
	}
	if got := resolveVoiceID("unknown"); got != "unknown" {
		t.Fatalf("unknown voice = %q", got)
	}
}
