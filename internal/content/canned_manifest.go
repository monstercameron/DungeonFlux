package content

import (
	"fmt"
	"strings"
)

// CannedManifest is the build-time asset manifest subset needed by canned
// lines. Its JSON shape matches scripts/buildtime/manifest.json.
type CannedManifest struct {
	Version int                    `json:"version"`
	Assets  map[string]CannedAsset `json:"assets"`
}

// CannedAsset describes the recorded audio for one logical canned line.
type CannedAsset struct {
	Kind       string       `json:"kind"`
	Selected   int          `json:"selected"`
	Takes      []CannedTake `json:"takes"`
	DurationMS int64        `json:"duration_ms,omitempty"`
}

// CannedTake identifies one content-addressed recording in the manifest.
type CannedTake struct {
	Number int    `json:"number"`
	SHA256 string `json:"sha256"`
	Path   string `json:"path"`
}

// ResolvedCannedLine pairs a fixed line with its selected manifest recording.
type ResolvedCannedLine struct {
	Line CannedLine
	Take CannedTake
}

// Resolve returns the selected recording for a logical canned line.
func (m CannedManifest) Resolve(id string) (CannedTake, bool) {
	asset, ok := m.Assets[id]
	if !ok || asset.Kind != "AUDIO" || asset.Selected < 1 {
		return CannedTake{}, false
	}
	for _, take := range asset.Takes {
		if take.Number == asset.Selected && validCannedTake(take) {
			return take, true
		}
	}
	return CannedTake{}, false
}

// ResolveAll returns every canned line paired with its selected recording in
// the stable order defined by CannedLines.
func (m CannedManifest) ResolveAll() ([]ResolvedCannedLine, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	resolved := make([]ResolvedCannedLine, 0, len(cannedLines))
	for _, line := range cannedLines {
		take, ok := m.Resolve(line.ID)
		if !ok {
			return nil, fmt.Errorf("canned manifest: %q has no selected audio take", line.ID)
		}
		resolved = append(resolved, ResolvedCannedLine{Line: line, Take: take})
	}
	return resolved, nil
}

// Validate checks that the manifest contains a usable recording for every
// canned line and does not contain malformed selected takes.
func (m CannedManifest) Validate() error {
	if m.Version < 1 {
		return fmt.Errorf("canned manifest: unsupported version %d", m.Version)
	}
	if m.Assets == nil {
		return fmt.Errorf("canned manifest: assets are missing")
	}
	for _, line := range cannedLines {
		asset, ok := m.Assets[line.ID]
		if !ok {
			return fmt.Errorf("canned manifest: missing %q", line.ID)
		}
		if err := validateCannedAsset(line.ID, asset); err != nil {
			return err
		}
	}
	return nil
}

func validateCannedAsset(id string, asset CannedAsset) error {
	if asset.Kind != "AUDIO" {
		return fmt.Errorf("canned manifest: %q has kind %q, want AUDIO", id, asset.Kind)
	}
	if asset.Selected < 1 {
		return fmt.Errorf("canned manifest: %q has no selected take", id)
	}
	seen := make(map[int]bool, len(asset.Takes))
	selected := false
	for _, take := range asset.Takes {
		if take.Number < 1 || seen[take.Number] || !validCannedTake(take) {
			return fmt.Errorf("canned manifest: %q has invalid take %d", id, take.Number)
		}
		seen[take.Number] = true
		selected = selected || take.Number == asset.Selected
	}
	if !selected {
		return fmt.Errorf("canned manifest: %q selects unavailable take %d", id, asset.Selected)
	}
	return nil
}

func validCannedTake(take CannedTake) bool {
	return take.Number > 0 && len(take.SHA256) == 64 && !strings.ContainsAny(take.SHA256, " \t\r\n") && strings.TrimSpace(take.Path) != ""
}
