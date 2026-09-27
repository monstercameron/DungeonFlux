package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
)

func TestPreparedBillboards_PersistAcrossRunsAndFollowClass(t *testing.T) {
	catalogue := []domain.Asset{
		preparedTestClip("paladin", "idle", "/assets/p-idle.mp4"),
		preparedTestClip("paladin", "attack", "/assets/p-attack.mp4"),
		preparedTestClip("rogue", "idle", "/assets/r-idle.mp4"),
	}
	hub := newBillboardHub(config.Config{}, catalogue, "/splat/scenes/battle.json")
	defer hub.cancel()
	inner := &fakes.FakeEngine{ViewValue: domain.View{Battlefield: &domain.BattlefieldView{Tokens: []domain.TokenView{
		{ID: "pc-1", Kind: "pc-rogue-human"},
		{ID: "pc-2", Kind: "pc-paladin-human"},
		{ID: "pc-3", Kind: "pc-wizard-human"},
	}}}}
	engine := newBillboardEngine(inner, hub)
	for run := range 2 {
		view := engine.View()
		if got := view.Battlefield.Tokens[0].Clips["idle"]; got != "/assets/r-idle.mp4" {
			t.Fatalf("run %d: seat 1 used the wrong hero: %q", run, got)
		}
		if got := view.Battlefield.Tokens[1].Clips["attack"]; got != "/assets/p-attack.mp4" {
			t.Fatalf("run %d: missing paladin attack: %q", run, got)
		}
		if len(view.Battlefield.Tokens[2].Clips) != 0 || inner.ViewValue.Battlefield.Tokens[0].Clips != nil {
			t.Fatal("cache substituted an unrelated hero or mutated engine state")
		}
		engine = newBillboardEngine(inner, hub)
	}
	engine.Step(domain.Envelope{Event: domain.AssetReady{Slot: "billboard:1:idle", Asset: domain.Asset{URL: "/assets/live.mp4"}}})
	if got := engine.View().Battlefield.Tokens[0].Clips["idle"]; got != "/assets/live.mp4" {
		t.Fatalf("live identity did not replace prepared default: %q", got)
	}
	engine = newBillboardEngine(inner, hub)
	if got := engine.View().Battlefield.Tokens[0].Clips["idle"]; got != "/assets/r-idle.mp4" {
		t.Fatalf("reset retained old live identity instead of prepared default: %q", got)
	}
	other := newBillboardHub(config.Config{}, catalogue, "/splat/scenes/other.json")
	defer other.cancel()
	if clips := other.decorate(inner.View()).Battlefield.Tokens[0].Clips; len(clips) != 0 {
		t.Fatal("prepared lighting/camera used for a different battlefield")
	}
}

func preparedTestClip(class, action, url string) domain.Asset {
	return domain.Asset{ID: domain.AssetID("hero_loop_" + class + "_" + action), URL: url,
		Meta: map[string]string{"class": class, "action": action, "battlefield_scene": "/splat/scenes/battle.json"}}
}

func TestLoadManifest_PreparedBillboardsRequireVerifiedContent(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		change                 func(*ManifestAsset)
		missing, corrupt, want bool
	}{
		{name: "verified", want: true},
		{name: "missing file", missing: true},
		{name: "corrupt bytes", corrupt: true},
		{name: "wrong action", change: func(e *ManifestAsset) { e.Metadata["action"] = "dance" }},
		{name: "wrong class", change: func(e *ManifestAsset) { e.Metadata["class"] = "rogue" }},
		{name: "missing battlefield", change: func(e *ManifestAsset) { delete(e.Metadata, "battlefield_scene") }},
		{name: "missing duration", change: func(e *ManifestAsset) { e.DurationMS = 0 }},
		{name: "unselected", change: func(e *ManifestAsset) { e.Selected = 0 }},
		{name: "wrong filename", change: func(e *ManifestAsset) { e.Takes[0].Path = "assets/other.mp4" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			data := []byte("cached video fixture")
			digest := sha256.Sum256(data)
			hash := hex.EncodeToString(digest[:])
			entry := ManifestAsset{Kind: "VIDEO", Selected: 1, DurationMS: 4000,
				Takes:    []ManifestTake{{Number: 1, SHA256: hash, Path: "assets/" + hash + ".mp4"}},
				Metadata: preparedTestClip("paladin", "idle", "").Meta}
			if tc.change != nil {
				tc.change(&entry)
			}
			if err := os.Mkdir(filepath.Join(root, "assets"), 0o700); err != nil {
				t.Fatal(err)
			}
			if !tc.missing {
				if tc.corrupt {
					data = []byte("damaged")
				}
				if err := os.WriteFile(filepath.Join(root, "assets", hash+".mp4"), data, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			manifest, err := json.Marshal(BuildManifest{Version: 1, Assets: map[string]ManifestAsset{"hero_loop_paladin_idle": entry}})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "manifest.json")
			if err := os.WriteFile(path, manifest, 0o600); err != nil {
				t.Fatal(err)
			}
			result, err := LoadManifest(path, nil)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, asset := range result.OneShot.Catalogue {
				if asset.ID == "hero_loop_paladin_idle" {
					found = true
					if asset.URL != "/assets/"+hash+".mp4" || asset.DurationMS != 4000 {
						t.Fatalf("asset=%#v", asset)
					}
				}
			}
			if found != tc.want {
				t.Fatalf("registered=%v, want %v", found, tc.want)
			}
		})
	}
}
