package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/media"
)

// applyPreparedBillboards registers local, content-verified hero loops. These
// are build-time defaults, not per-run results, so a reset cannot erase them.
func applyPreparedBillboards(story *content.OneShot, manifest BuildManifest, root string) {
	for name, entry := range manifest.Assets {
		if !strings.HasPrefix(name, "hero_loop_") || entry.DurationMS <= 0 {
			continue
		}
		class, action := entry.Metadata["class"], entry.Metadata["action"]
		if class == "" || !media.BillboardActions(action) || name != "hero_loop_"+class+"_"+action || entry.Metadata["battlefield_scene"] == "" {
			continue
		}
		take, ok := selectedTake(entry)
		if !ok || filepath.Base(take.Path) != take.SHA256+".mp4" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, "assets", filepath.Base(take.Path)))
		hash := sha256.Sum256(data)
		if err != nil || len(data) == 0 || hex.EncodeToString(hash[:]) != take.SHA256 {
			continue
		}
		story.Catalogue = append(story.Catalogue, domain.Asset{
			ID: domain.AssetID(name), URL: assetURL(root, take.Path), SHA256: take.SHA256,
			Kind: "VIDEO", MIME: "video/mp4", DurationMS: int(entry.DurationMS), Meta: cloneStrings(entry.Metadata),
		})
	}
}

func preparedHeroClips(catalogue []domain.Asset, sceneURL string) map[string]map[string]string {
	clips := make(map[string]map[string]string)
	for _, asset := range catalogue {
		class, action := asset.Meta["class"], asset.Meta["action"]
		if class == "" || asset.URL == "" || !media.BillboardActions(action) || asset.Meta["battlefield_scene"] != sceneURL || string(asset.ID) != "hero_loop_"+class+"_"+action {
			continue
		}
		if clips[class] == nil {
			clips[class] = make(map[string]string)
		}
		clips[class][action] = asset.URL
	}
	return clips
}

func (h *billboardHub) preparedClipsFor(token domain.TokenView) map[string]string {
	if !strings.HasPrefix(token.Kind, "pc-") {
		return nil
	}
	class, _, _ := strings.Cut(strings.TrimPrefix(token.Kind, "pc-"), "-")
	return h.prepared[class]
}
