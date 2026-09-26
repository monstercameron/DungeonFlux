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

func TestSFXAssets_CoversGeneralAndCombatLibrary(t *testing.T) {
	assets := SFXAssets()
	if len(assets) != 16 {
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
	data, err := BuildSFXRequest(SFXAssets()[0])
	if err != nil {
		t.Fatal(err)
	}
	var request SFXRequest
	if err := json.Unmarshal(data, &request); err != nil || request.Text == "" {
		t.Fatalf("invalid request: %v %#v", err, request)
	}
	if _, err := BuildSFXRequest(SFXAsset{ID: "bad", Prompt: "sound", LUFS: 0}); err == nil {
		t.Fatal("accepted non-negative loudness")
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
	asset := SFXAssets()[0]
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
