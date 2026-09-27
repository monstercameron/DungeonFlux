package content

import "testing"

func TestCannedManifest_ResolvesEverySpecLine(t *testing.T) {
	manifest := cannedManifestFixture()
	resolved, err := manifest.ResolveAll()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(resolved), len(CannedLines()); got != want {
		t.Fatalf("resolved %d lines, want %d", got, want)
	}
	for _, item := range resolved {
		if item.Take.Number != 1 || item.Take.Path == "" || item.Line.ID == "" {
			t.Errorf("incomplete resolution: %+v", item)
		}
	}
}

func TestCannedManifest_RejectsInvalidFixtures(t *testing.T) {
	tests := []struct {
		name   string
		change func(*CannedManifest)
	}{
		{name: "missing line", change: func(m *CannedManifest) { delete(m.Assets, CannedOpeningID) }},
		{name: "wrong kind", change: func(m *CannedManifest) {
			asset := m.Assets[CannedOpeningID]
			asset.Kind = "IMAGE"
			m.Assets[CannedOpeningID] = asset
		}},
		{name: "unavailable selection", change: func(m *CannedManifest) {
			asset := m.Assets[CannedOpeningID]
			asset.Selected = 2
			m.Assets[CannedOpeningID] = asset
		}},
		{name: "invalid hash", change: func(m *CannedManifest) {
			asset := m.Assets[CannedOpeningID]
			asset.Takes[0].SHA256 = "short"
			m.Assets[CannedOpeningID] = asset
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			manifest := cannedManifestFixture()
			tc.change(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate accepted malformed manifest")
			}
		})
	}
}

func TestCannedManifest_ResolveRejectsUnknownOrNonAudio(t *testing.T) {
	manifest := cannedManifestFixture()
	if _, ok := manifest.Resolve("missing"); ok {
		t.Fatal("Resolve found an unknown line")
	}
	asset := manifest.Assets[CannedOpeningID]
	asset.Kind = "IMAGE"
	manifest.Assets[CannedOpeningID] = asset
	if _, ok := manifest.Resolve(CannedOpeningID); ok {
		t.Fatal("Resolve found a non-audio line")
	}
}

func cannedManifestFixture() CannedManifest {
	assets := make(map[string]CannedAsset, len(CannedLines()))
	for _, line := range CannedLines() {
		assets[line.ID] = CannedAsset{Kind: "AUDIO", Selected: 1, Takes: []CannedTake{{
			Number: 1,
			SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Path:   "assets/" + line.ID + ".pcm",
		}}}
	}
	return CannedManifest{Version: 1, Assets: assets}
}
