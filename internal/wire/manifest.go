package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// BuildManifest is the runtime representation of the build-time asset index.
type BuildManifest struct {
	Version int                      `json:"version"`
	Assets  map[string]ManifestAsset `json:"assets"`
}

// ManifestAsset identifies the selected take and its measured media metadata.
type ManifestAsset struct {
	Kind       string            `json:"kind"`
	Selected   int               `json:"selected"`
	Takes      []ManifestTake    `json:"takes"`
	DurationMS int64             `json:"duration_ms"`
	ContactMS  int64             `json:"contact_ms"`
	Metadata   map[string]string `json:"metadata"`
}

// ManifestTake identifies one content-addressed build-time file.
type ManifestTake struct {
	Number int    `json:"number"`
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
}

// ManifestResult contains the resolved one-shot and manifest digest.
type ManifestResult struct {
	OneShot content.OneShot
	Hash    string
}

// LoadManifest reads a build-time manifest and applies selected assets to the
// default one-shot. Missing or unavailable assets retain their logical names.
func LoadManifest(path string, logger *slog.Logger) (ManifestResult, error) {
	story := content.DefaultOneShot()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		warn(logger, "build-time manifest missing", "path", path)
		return ManifestResult{OneShot: story}, nil
	}
	if err != nil {
		return ManifestResult{}, fmt.Errorf("read build-time manifest: %w", err)
	}
	digest := sha256.Sum256(data)
	var manifest BuildManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return ManifestResult{}, fmt.Errorf("decode build-time manifest: %w", err)
	}
	if manifest.Version < 1 {
		return ManifestResult{}, fmt.Errorf("build-time manifest: unsupported version %d", manifest.Version)
	}
	applyManifest(&story, manifest, filepath.Dir(path), logger)
	applyKillcams(&story, manifest, filepath.Dir(path))
	return ManifestResult{OneShot: story, Hash: hex.EncodeToString(digest[:])}, nil
}

func applyManifest(story *content.OneShot, manifest BuildManifest, root string, logger *slog.Logger) {
	assets := make(map[string]domain.Asset, len(story.Catalogue)+len(content.CannedLines()))
	for _, asset := range story.Catalogue {
		assets[string(asset.ID)] = asset
	}
	for _, line := range content.CannedLines() {
		assets[line.ID] = domain.Asset{ID: domain.AssetID(line.ID), Kind: "AUDIO"}
	}
	for id, asset := range assets {
		entry, ok := manifest.Assets[id]
		if !ok {
			warn(logger, "build-time asset missing; using logical name", "asset", id)
			continue
		}
		take, ok := selectedTake(entry)
		if !ok {
			warn(logger, "build-time asset has no selected take; using logical name", "asset", id)
			continue
		}
		asset.SHA256 = take.SHA256
		asset.URL = assetURL(root, take.Path)
		asset.DurationMS = int(entry.DurationMS)
		asset.Meta = cloneStrings(entry.Metadata)
		if entry.ContactMS > 0 {
			if asset.Meta == nil {
				asset.Meta = make(map[string]string)
			}
			asset.Meta["contact_ms"] = fmt.Sprint(entry.ContactMS)
		}
		assets[id] = asset
	}
	story.Catalogue = make([]domain.Asset, 0, len(assets))
	for _, asset := range assets {
		story.Catalogue = append(story.Catalogue, asset)
	}
}

func selectedTake(asset ManifestAsset) (ManifestTake, bool) {
	if asset.Selected < 1 {
		return ManifestTake{}, false
	}
	for _, take := range asset.Takes {
		if take.Number == asset.Selected && len(take.SHA256) == 64 && strings.TrimSpace(take.Path) != "" {
			return take, true
		}
	}
	return ManifestTake{}, false
}

func assetURL(root, path string) string {
	_ = root
	return "/assets/" + filepath.ToSlash(filepath.Base(path))
}

func cloneStrings(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	output := make(map[string]string, len(input))
	maps.Copy(output, input)
	return output
}

func warn(logger *slog.Logger, message string, args ...any) {
	if logger != nil {
		logger.Warn(message, args...)
	}
}
