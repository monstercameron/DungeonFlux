package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSFXAssets_CoversGeneralAndCombatLibrary(t *testing.T) {
	assets := SFXAssets()
	if len(assets) != 24 {
		t.Fatalf("got %d SFX assets", len(assets))
	}
	seen := make(map[string]bool)
	for _, asset := range assets {
		if asset.ID == "" || asset.Prompt == "" || asset.LUFS >= 0 || seen[asset.ID] {
			t.Fatalf("invalid or duplicate SFX: %#v", asset)
		}
		seen[asset.ID] = true
	}
	for _, id := range []string{"sfx_dice_roll", "sfx_tavern_ambience", "sfx_sword_slash", "sfx_wet_footsteps"} {
		if !seen[id] {
			t.Fatalf("missing required SFX %q", id)
		}
	}
}

func TestBuildSFXRequest_EncodesPrompt(t *testing.T) {
	data, err := BuildSFXRequest(SFXAssets()[8])
	if err != nil {
		t.Fatal(err)
	}
	var request SFXRequest
	if err := json.Unmarshal(data, &request); err != nil || request.Text == "" {
		t.Fatalf("invalid request: %v %#v", err, request)
	}
	if request.ModelID != "eleven_text_to_sound_v2" || request.DurationSeconds != 2 {
		t.Fatalf("unexpected request settings: %#v", request)
	}
	ambience, err := BuildSFXRequest(SFXAssets()[12])
	if err != nil {
		t.Fatal(err)
	}
	var ambienceRequest SFXRequest
	if err := json.Unmarshal(ambience, &ambienceRequest); err != nil || !ambienceRequest.Loop {
		t.Fatalf("ambience loop was not encoded: %v %#v", err, ambienceRequest)
	}
	if _, err := BuildSFXRequest(SFXAsset{ID: "bad", Prompt: "sound", LUFS: 0}); err == nil {
		t.Fatal("accepted non-negative loudness")
	}
	if _, err := BuildSFXRequest(SFXAsset{ID: "bad", Prompt: "sound", LUFS: -16, DurationSeconds: 31}); err == nil {
		t.Fatal("accepted invalid duration")
	}
}

func TestRenderSFX_StoresAssetAndNormalizationMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sound-generation" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("effect"))
	}))
	defer server.Close()
	root := t.TempDir()
	manifestRoot := filepath.Join(root, "manifest")
	writer, err := NewManifestWriter(manifestRoot)
	if err != nil {
		t.Fatal(err)
	}
	asset := SFXAssets()[8]
	if err := RenderSFX(context.Background(), server.Client(), server.URL, root, writer, asset, 1); err != nil {
		t.Fatal(err)
	}
	stored := writer.manifest.Assets[asset.ID]
	if stored.Metadata["target_lufs"] != "-16" || stored.Metadata["postprocess"] == "" {
		t.Fatalf("normalization metadata missing: %#v", stored.Metadata)
	}
	data, err := os.ReadFile(filepath.Join(manifestRoot, stored.Takes[0].Path))
	if err != nil || string(data) != "effect" {
		t.Fatalf("stored effect mismatch: %v %q", err, data)
	}
}

type fakeSFXProcessor struct {
	stats SFXMediaStats
}

func (p fakeSFXProcessor) Normalize(_ context.Context, source, destination string, target int) (SFXMediaStats, error) {
	data, err := os.ReadFile(source)
	if err != nil {
		return SFXMediaStats{}, err
	}
	if err := os.WriteFile(destination, append([]byte("normalized:"), data...), 0o644); err != nil {
		return SFXMediaStats{}, err
	}
	stats := p.stats
	stats.IntegratedLUFS = float64(target)
	return stats, nil
}

func TestRenderSFXWithProcessor_RegistersNormalizedTake(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sound-generation" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = io.WriteString(w, "raw-effect")
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	asset := SFXAssets()[0]
	stats, err := RenderSFXWithProcessor(context.Background(), server.Client(), server.URL, filepath.Join(root, "sfx"), writer, asset, 2, fakeSFXProcessor{stats: SFXMediaStats{DurationSeconds: 2.1, IntegratedLUFS: -16.2}})
	if err != nil {
		t.Fatal(err)
	}
	if stats.DurationSeconds != 2.1 || writer.manifest.Assets[asset.ID].DurationMS != 2100 {
		t.Fatalf("unexpected stats or metadata: %#v %#v", stats, writer.manifest.Assets[asset.ID])
	}
	stored := writer.manifest.Assets[asset.ID].Takes[0]
	data, err := os.ReadFile(filepath.Join(root, stored.Path))
	if err != nil || !strings.Contains(string(data), "normalized:raw-effect") {
		t.Fatalf("normalized asset mismatch: %v %q", err, data)
	}
}

func TestPlanSFX_AccountsForTwoTakes(t *testing.T) {
	plan, err := PlanSFX(SFXAssets(), 2)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Requests != 48 || plan.EstimatedSeconds != 103.2 || plan.EstimatedCostUSD <= 0 {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if _, err := PlanSFX(SFXAssets(), 4); err == nil {
		t.Fatal("accepted more than three takes")
	}
}

func TestRunSFXBuild_SelectsBestTakeAndReleasesLock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "raw-effect")
	}))
	defer server.Close()
	root := t.TempDir()
	processor := fakeSFXProcessor{stats: SFXMediaStats{DurationSeconds: 2, IntegratedLUFS: -16}}
	var log strings.Builder
	summary, err := RunSFXBuild(context.Background(), SFXBuildOptions{Client: server.Client(), Endpoint: server.URL, Root: root, Takes: 1, Processor: processor, Log: &log})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Requests != len(SFXAssets()) || summary.Selected != len(SFXAssets()) {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !os.IsNotExist(err) {
		t.Fatalf("manifest lock remains: %v", err)
	}
	if !strings.Contains(log.String(), "sfx_summary requests=24") {
		t.Fatalf("summary log missing: %s", log.String())
	}
}

func TestSFXBuild_RejectsInvalidInputsAndHonorsCancelledLockWait(t *testing.T) {
	if _, err := RunSFXBuild(context.Background(), SFXBuildOptions{}); err == nil {
		t.Fatal("accepted empty build options")
	}
	root := t.TempDir()
	lockPath := filepath.Join(root, "manifest.lock")
	if err := os.WriteFile(lockPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := acquireSFXManifestLock(ctx, root); err == nil {
		t.Fatal("accepted cancelled lock wait")
	}
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
}

func TestSFXBuild_RejectsBadStatsAndHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "raw-effect")
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	bad := fakeSFXProcessor{stats: SFXMediaStats{DurationSeconds: 0.1, IntegratedLUFS: -16}}
	if _, err := RenderSFXWithProcessor(context.Background(), server.Client(), server.URL, filepath.Join(root, "sfx"), writer, SFXAssets()[0], 1, bad); err == nil {
		t.Fatal("accepted an effect with invalid duration")
	}
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer errorServer.Close()
	if _, err := requestSFX(context.Background(), errorServer.Client(), errorServer.URL, root, SFXAssets()[0]); err == nil {
		t.Fatal("accepted unauthorized SFX response")
	}
	if _, err := PlanSFX([]SFXAsset{{ID: "bad", Prompt: ""}}, 2); err == nil {
		t.Fatal("accepted an invalid SFX plan")
	}
}

func TestSFXBuild_RejectsCostCapBeforeNetwork(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("cost cap should reject before making a request")
	}))
	defer server.Close()
	_, err := RunSFXBuild(context.Background(), SFXBuildOptions{Client: server.Client(), Endpoint: server.URL, Root: t.TempDir(), Takes: 3, MaxCostUSD: 0.001})
	if err == nil {
		t.Fatal("accepted a plan over the cost cap")
	}
}

func TestFilterSFXAssetsKeepsOnlyNamedAssets(t *testing.T) {
	all := SFXAssets()
	if got := filterSFXAssets(all, nil); len(got) != len(all) {
		t.Fatalf("empty filter kept %d of %d", len(got), len(all))
	}
	got := filterSFXAssets(all, []string{"sfx_hero_lock", " sfx_join_tv", "missing"})
	if len(got) != 2 {
		t.Fatalf("filter kept %d assets, want 2", len(got))
	}
	for _, asset := range got {
		if asset.ID != "sfx_hero_lock" && asset.ID != "sfx_join_tv" {
			t.Fatalf("unexpected asset %q", asset.ID)
		}
	}
}
