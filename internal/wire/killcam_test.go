package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/content"
)

func TestApplyKillcams_VerifiesFilesBeforeEnablingPlayback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0700); err != nil {
		t.Fatal(err)
	}
	data := []byte("video")
	hash := sha256.Sum256(data)
	sha := hex.EncodeToString(hash[:])
	path := filepath.Join(root, "assets", sha+".mp4")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	entry := ManifestAsset{Selected: 1, DurationMS: 4000, Metadata: map[string]string{"class": "paladin", "outcome": "victory"}, Takes: []ManifestTake{{Number: 1, SHA256: sha, Path: "assets/" + sha + ".mp4"}}}
	for _, tc := range []struct {
		name  string
		edit  func(*ManifestAsset)
		valid bool
	}{
		{"valid", func(*ManifestAsset) {}, true},
		{"long", func(e *ManifestAsset) { e.DurationMS = 9000 }, false},
		{"missing", func(e *ManifestAsset) { e.Selected = 2 }, false},
		{"bad outcome", func(e *ManifestAsset) { e.Metadata = map[string]string{"class": "paladin", "outcome": "fled"} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := entry
			tc.edit(&e)
			story := content.DefaultOneShot()
			applyKillcams(&story, BuildManifest{Assets: map[string]ManifestAsset{"killcam_paladin_victory": e}}, root)
			_, ok := story.KillCams["paladin_victory"]
			if ok != tc.valid {
				t.Fatalf("enabled=%v", ok)
			}
		})
	}
	if err := os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	story := content.DefaultOneShot()
	applyKillcams(&story, BuildManifest{Assets: map[string]ManifestAsset{"killcam_paladin_victory": entry}}, root)
	if len(story.KillCams) != 0 {
		t.Fatal("corrupt video enabled")
	}
}
