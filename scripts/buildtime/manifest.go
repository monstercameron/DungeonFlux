// Package main contains the build-time asset runner and manifest writer.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const manifestVersion = 1

// Manifest is the deterministic index of build-time assets.
type Manifest struct {
	Version int              `json:"version"`
	Assets  map[string]Asset `json:"assets"`
}

// Asset identifies a logical asset and its available takes.
type Asset struct {
	Kind       string            `json:"kind,omitempty"`
	Selected   int               `json:"selected"`
	Takes      []Take            `json:"takes"`
	DurationMS int64             `json:"duration_ms,omitempty"`
	ContactMS  int64             `json:"contact_ms,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Take is one content-addressed build-time asset file.
type Take struct {
	Number int    `json:"number"`
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
}

// ManifestWriter adds content-addressed files to a build-time manifest.
type ManifestWriter struct {
	root     string
	manifest Manifest
	mu       sync.Mutex
}

// NewManifestWriter loads an existing manifest, or creates an empty one.
func NewManifestWriter(root string) (*ManifestWriter, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("buildtime: empty output directory")
	}
	w := &ManifestWriter{root: filepath.Clean(root), manifest: Manifest{
		Version: manifestVersion,
		Assets:  make(map[string]Asset),
	}}
	data, err := os.ReadFile(filepath.Join(w.root, "manifest.json"))
	if errors.Is(err, os.ErrNotExist) {
		return w, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if err := json.Unmarshal(data, &w.manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	if w.manifest.Version != manifestVersion {
		return nil, fmt.Errorf("unsupported manifest version %d", w.manifest.Version)
	}
	if w.manifest.Assets == nil {
		w.manifest.Assets = make(map[string]Asset)
	}
	return w, nil
}

// AddFile hashes source, copies it into the build-time asset store, and records a take.
func (w *ManifestWriter) AddFile(logical, kind, source string, takeNumber int) (Take, error) {
	if err := validateLogicalName(logical); err != nil {
		return Take{}, err
	}
	if takeNumber < 1 {
		return Take{}, errors.New("buildtime: take number must be positive")
	}
	if strings.TrimSpace(kind) == "" {
		return Take{}, errors.New("buildtime: asset kind is required")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return Take{}, fmt.Errorf("read asset %q: %w", source, err)
	}
	hash := sha256.Sum256(data)
	sha := hex.EncodeToString(hash[:])
	ext := filepath.Ext(source)
	if ext == "" {
		ext = ".bin"
	}
	name := sha + ext
	assetDir := filepath.Join(w.root, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		return Take{}, fmt.Errorf("create asset directory: %w", err)
	}
	destination := filepath.Join(assetDir, name)
	if err := writeIfAbsent(destination, data); err != nil {
		return Take{}, fmt.Errorf("store asset: %w", err)
	}
	take := Take{Number: takeNumber, SHA256: sha, Path: filepath.ToSlash(filepath.Join("assets", name))}
	w.mu.Lock()
	defer w.mu.Unlock()
	asset := w.manifest.Assets[logical]
	asset.Kind = kind
	asset.Takes = replaceTake(asset.Takes, take)
	if asset.Selected == 0 {
		asset.Selected = 1
	}
	w.manifest.Assets[logical] = asset
	return take, nil
}

// SetMetadata updates media metadata for a logical asset.
func (w *ManifestWriter) SetMetadata(logical string, durationMS, contactMS int64, metadata map[string]string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	asset, ok := w.manifest.Assets[logical]
	if !ok {
		return fmt.Errorf("buildtime: asset %q not found", logical)
	}
	if durationMS < 0 || contactMS < 0 {
		return errors.New("buildtime: media timings cannot be negative")
	}
	asset.DurationMS, asset.ContactMS = durationMS, contactMS
	if metadata != nil {
		asset.Metadata = cloneMetadata(metadata)
	}
	w.manifest.Assets[logical] = asset
	return nil
}

// SelectTake selects the take used by consumers while retaining all takes.
func (w *ManifestWriter) SelectTake(logical string, number int) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	asset, ok := w.manifest.Assets[logical]
	if !ok {
		return fmt.Errorf("buildtime: asset %q not found", logical)
	}
	for _, take := range asset.Takes {
		if take.Number == number {
			asset.Selected = number
			w.manifest.Assets[logical] = asset
			return nil
		}
	}
	return fmt.Errorf("buildtime: take %d for %q not found", number, logical)
}

// Write persists the manifest atomically and returns its path.
func (w *ManifestWriter) Write() (string, error) {
	w.mu.Lock()
	data, err := json.MarshalIndent(w.manifest, "", "  ")
	w.mu.Unlock()
	if err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	if err := os.MkdirAll(w.root, 0o755); err != nil {
		return "", fmt.Errorf("create buildtime directory: %w", err)
	}
	path := filepath.Join(w.root, "manifest.json")
	tmp, err := os.CreateTemp(w.root, ".manifest-*.json")
	if err != nil {
		return "", fmt.Errorf("create manifest temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return "", fmt.Errorf("write manifest: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close manifest: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("replace manifest: %w", err)
	}
	return path, nil
}

func validateLogicalName(name string) error {
	if strings.TrimSpace(name) == "" || filepath.Base(name) != name || strings.Contains(name, "\\") {
		return fmt.Errorf("buildtime: invalid logical name %q", name)
	}
	return nil
}

func writeIfAbsent(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func replaceTake(takes []Take, replacement Take) []Take {
	for i, take := range takes {
		if take.Number == replacement.Number {
			takes[i] = replacement
			return sortedTakes(takes)
		}
	}
	return sortedTakes(append(takes, replacement))
}

func sortedTakes(takes []Take) []Take {
	sort.Slice(takes, func(i, j int) bool { return takes[i].Number < takes[j].Number })
	return takes
}

func cloneMetadata(input map[string]string) map[string]string {
	output := make(map[string]string, len(input))
	for key, value := range input {
		output[key] = value
	}
	return output
}
