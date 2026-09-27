package wire

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestNewExecutors_ReferenceEffectRunsReferenceAndVoicePack(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.Config{Server: config.ServerConfig{DataDir: dataDir}, Adapters: map[string]config.AdapterConfig{
		"sound": {Vendor: "elevenlabs", Mode: "fake"},
	}, Budget: config.BudgetConfig{HardUSD: 2}}
	runner, _, err := newExecutors(configForWire{config: cfg}, nil)
	if err != nil {
		t.Fatalf("newExecutors() error = %v", err)
	}
	runner.Run([]domain.Effect{domain.GenerateCharacterReference{Seat: 1, Species: "elf", Class: "wizard"}})
	deadline := time.NewTimer(2 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for {
		entries, readErr := os.ReadDir(filepath.Join(dataDir, "assets"))
		if readErr == nil && len(entries) >= 7 {
			settled := time.NewTimer(100 * time.Millisecond)
			<-settled.C
			return
		}
		select {
		case <-deadline.C:
			names := make([]string, 0, len(entries))
			for _, entry := range entries {
				names = append(names, entry.Name())
			}
			t.Fatalf("reference and voice-pack assets were not written; count=%d names=%v read error = %v", len(entries), names, readErr)
		case <-ticker.C:
		}
	}
}

func TestReferenceFallback_UsesManifestSpeciesAndClassArt(t *testing.T) {
	t.Chdir(t.TempDir())
	root := filepath.Dir(manifestPath())
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	manifest := BuildManifest{Version: 1, Assets: make(map[string]ManifestAsset)}
	for _, fixture := range []struct {
		name  string
		id    string
		color color.RGBA
	}{
		{"species.png", "ui/species_elf", color.RGBA{R: 255, A: 255}}, {"class.png", "ui/class_wizard", color.RGBA{B: 255, A: 255}},
	} {
		canvas := image.NewRGBA(image.Rect(0, 0, 2, 3))
		canvas.SetRGBA(0, 0, fixture.color)
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, canvas); err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(encoded.Bytes())
		manifest.Assets[fixture.id] = ManifestAsset{Selected: 1, Takes: []ManifestTake{{Number: 1, Path: fixture.name, SHA256: hex.EncodeToString(hash[:])}}}
		if err := os.WriteFile(filepath.Join(root, fixture.name), encoded.Bytes(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath(), data, 0600); err != nil {
		t.Fatal(err)
	}
	sheet, _ := referenceFallback(domain.GenerateCharacterReference{Species: "elf", Class: "wizard"})
	got, err := png.Decode(bytes.NewReader(sheet))
	if err != nil {
		t.Fatalf("decode composed manifest art: %v", err)
	}
	if got.Bounds().Dx() != 16 || got.Bounds().Dy() != 3 {
		t.Fatalf("sheet bounds = %v", got.Bounds())
	}
	for panel := range 4 {
		if color.RGBAModel.Convert(got.At(panel*4, 0)) != (color.RGBA{R: 255, A: 255}) || color.RGBAModel.Convert(got.At(panel*4+2, 0)) != (color.RGBA{B: 255, A: 255}) {
			t.Fatalf("panel %d lost species/class art", panel)
		}
	}
}
