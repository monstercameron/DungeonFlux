package wire

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/api"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type assetCatalog struct {
	byName map[string]assetRecord
	bySHA  map[string]assetRecord
}

type assetRecord struct {
	info api.AssetInfo
	path string
}

// loadAssetCatalog copies selected manifest files into the runtime asset store
// and records their metadata for the gRPC service and SQLite asset index.
func loadAssetCatalog(ctx context.Context, dataDir, manifestPath string, stored ports.Assets, logger *slog.Logger) (*assetCatalog, error) {
	catalog := &assetCatalog{byName: make(map[string]assetRecord), bySHA: make(map[string]assetRecord)}
	data, err := os.ReadFile(manifestPath)
	if errors.Is(err, os.ErrNotExist) {
		return catalog, nil
	}
	if err != nil {
		return nil, fmt.Errorf("wire: read asset manifest: %w", err)
	}
	var manifest BuildManifest
	if err := unmarshalBuildManifest(data, &manifest); err != nil {
		return nil, err
	}
	root := filepath.Dir(manifestPath)
	destinationRoot := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(destinationRoot, 0o700); err != nil {
		return nil, fmt.Errorf("wire: create runtime asset store: %w", err)
	}
	keys := make([]string, 0, len(manifest.Assets))
	for name := range manifest.Assets {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		record, err := copyManifestAsset(root, destinationRoot, name, manifest.Assets[name])
		if err != nil {
			return nil, err
		}
		catalog.byName[name] = record
		catalog.bySHA[record.info.SHA256] = record
		if stored != nil {
			asset := domain.Asset{ID: domain.AssetID(name), SHA256: record.info.SHA256, Kind: manifest.Assets[name].Kind,
				MIME: record.info.ContentType, URL: "/assets/" + filepath.Base(record.path), DurationMS: int(manifest.Assets[name].DurationMS),
				Meta: cloneStrings(manifest.Assets[name].Metadata)}
			if err := stored.Put(ctx, asset); err != nil {
				return nil, fmt.Errorf("wire: index asset %q: %w", name, err)
			}
		}
	}
	if logger != nil && len(keys) > 0 {
		logger.Info("assets loaded", "count", len(keys), "manifest", manifestPath)
	}
	return catalog, nil
}

func unmarshalBuildManifest(data []byte, manifest *BuildManifest) error {
	if err := json.Unmarshal(data, manifest); err != nil {
		return fmt.Errorf("wire: decode asset manifest: %w", err)
	}
	if manifest.Version < 1 {
		return fmt.Errorf("wire: unsupported asset manifest version %d", manifest.Version)
	}
	return nil
}

func copyManifestAsset(root, destinationRoot, name string, entry ManifestAsset) (assetRecord, error) {
	take, ok := selectedTake(entry)
	if !ok {
		return assetRecord{}, fmt.Errorf("wire: asset %q has no valid selected take", name)
	}
	source, err := safeManifestPath(root, take.Path)
	if err != nil {
		return assetRecord{}, fmt.Errorf("wire: asset %q: %w", name, err)
	}
	info, err := os.Lstat(source)
	if err != nil {
		return assetRecord{}, fmt.Errorf("wire: stat asset %q: %w", name, err)
	}
	if !info.Mode().IsRegular() {
		return assetRecord{}, fmt.Errorf("wire: asset %q is not a regular file", name)
	}
	destination := filepath.Join(destinationRoot, filepath.Base(source))
	if err := copyAssetFile(source, destination); err != nil {
		return assetRecord{}, fmt.Errorf("wire: copy asset %q: %w", name, err)
	}
	return assetRecord{info: api.AssetInfo{Name: name, SHA256: take.SHA256, ContentType: assetContentType(entry.Kind, source), Size: info.Size()}, path: destination}, nil
}

func safeManifestPath(root, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	if filepath.IsAbs(clean) {
		return "", errors.New("path escapes build-time asset store")
	}
	path := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes build-time asset store")
	}
	return path, nil
}

func copyAssetFile(source, destination string) error {
	if filepath.Clean(source) == filepath.Clean(destination) {
		return nil
	}
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func assetContentType(kind, path string) string {
	if value := mime.TypeByExtension(filepath.Ext(path)); value != "" {
		return value
	}
	switch strings.ToUpper(kind) {
	case "IMAGE":
		return "image/*"
	case "AUDIO":
		return "audio/*"
	case "VIDEO", "CLIP":
		return "video/*"
	default:
		return "application/octet-stream"
	}
}

func (c *assetCatalog) Manifest(context.Context) ([]api.AssetInfo, error) {
	if c == nil {
		return nil, errors.New("wire: nil asset catalog")
	}
	assets := make([]api.AssetInfo, 0, len(c.byName))
	for _, record := range c.byName {
		assets = append(assets, record.info)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Name < assets[j].Name })
	return assets, nil
}

func (c *assetCatalog) Open(ctx context.Context, selector string) (api.AssetInfo, io.ReadCloser, error) {
	if c == nil {
		return api.AssetInfo{}, nil, errors.New("wire: nil asset catalog")
	}
	if err := ctx.Err(); err != nil {
		return api.AssetInfo{}, nil, err
	}
	record, ok := c.byName[selector]
	if !ok {
		record, ok = c.bySHA[selector]
	}
	if !ok {
		return api.AssetInfo{}, nil, os.ErrNotExist
	}
	reader, err := os.Open(record.path)
	if err != nil {
		return api.AssetInfo{}, nil, err
	}
	return record.info, reader, nil
}
