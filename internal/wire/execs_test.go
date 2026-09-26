package wire

import (
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
	t.Chdir(filepath.Join("..", ".."))
	if _, err := os.Stat(manifestPath()); err != nil {
		t.Skipf("build-time manifest unavailable: %v", err)
	}
	sheet, _ := referenceFallback(domain.GenerateCharacterReference{Species: "elf", Class: "wizard"})
	if len(sheet) == 0 {
		t.Fatal("reference fallback did not compose manifest art")
	}
}
