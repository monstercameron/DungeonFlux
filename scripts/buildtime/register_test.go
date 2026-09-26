package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRegisterScannedStills_RegistersKnownFiles(t *testing.T) {
	root := t.TempDir()
	for _, asset := range stillRegistry {
		if err := os.WriteFile(filepath.Join(root, asset.file), []byte(asset.file), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	path, err := RegisterScannedStills(root)
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "manifest.json") {
		t.Fatalf("manifest path = %q", path)
	}
	manifest := readTestManifest(t, path)
	if len(manifest.Assets) != len(stillRegistry) {
		t.Fatalf("registered %d assets, want %d", len(manifest.Assets), len(stillRegistry))
	}
	for _, want := range stillRegistry {
		asset, ok := manifest.Assets[want.logical]
		if !ok || asset.Kind != want.kind || asset.Selected != 1 || len(asset.Takes) != 1 {
			t.Errorf("manifest[%q] = %#v", want.logical, asset)
		}
	}
}

func TestRegisterScannedStills_SkipsMissingAndKeepsExistingTake(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tavern_interior.png"), []byte("first"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterScannedStills(root); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "tavern_interior.png"), []byte("second"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := RegisterScannedStills(root); err != nil {
		t.Fatal(err)
	}
	manifest := readTestManifest(t, filepath.Join(root, "manifest.json"))
	if len(manifest.Assets) != 1 || len(manifest.Assets["establishing_tavern"].Takes) != 1 {
		t.Fatalf("manifest after rescan = %#v", manifest.Assets)
	}
	if _, err := os.Stat(filepath.Join(root, "assets", manifest.Assets["establishing_tavern"].Takes[0].SHA256+".png")); err != nil {
		t.Fatal(err)
	}
}

func TestRunRegister_RequiresScan(t *testing.T) {
	if err := runRegister([]string{"--root", t.TempDir()}); err == nil {
		t.Fatal("runRegister accepted missing --scan")
	}
}

func readTestManifest(t *testing.T, path string) Manifest {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}
