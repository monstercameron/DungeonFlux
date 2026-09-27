package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// Only verified local clips enter the engine catalogue. Missing or corrupt
// media must never introduce a pause into a live fight.
func applyKillcams(story *content.OneShot, manifest BuildManifest, root string) {
	story.KillCams = make(map[string]domain.Asset)
	for name, entry := range manifest.Assets {
		if !strings.HasPrefix(name, "killcam_") || entry.DurationMS != 4000 {
			continue
		}
		class, outcome := entry.Metadata["class"], entry.Metadata["outcome"]
		if class == "" || (outcome != "victory" && outcome != "defeat") || name != "killcam_"+class+"_"+outcome {
			continue
		}
		take, ok := selectedTake(entry)
		if !ok || filepath.Ext(take.Path) != ".mp4" || filepath.Base(take.Path) != take.SHA256+".mp4" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "assets", filepath.Base(take.Path)))
		hash := sha256.Sum256(data)
		if err != nil || len(data) == 0 || hex.EncodeToString(hash[:]) != take.SHA256 {
			continue
		}
		asset := domain.Asset{ID: domain.AssetID(name), URL: assetURL(root, take.Path), SHA256: take.SHA256, Kind: "VIDEO", DurationMS: 4000, Meta: cloneStrings(entry.Metadata)}
		story.KillCams[class+"_"+outcome] = asset
		story.Catalogue = append(story.Catalogue, asset)
	}
}
