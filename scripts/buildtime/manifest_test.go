package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestManifestWriter_AddFileAndWrite_IsContentAddressed(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "opening.wav")
	if err := os.WriteFile(source, []byte{0, 1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := NewManifestWriter(filepath.Join(root, "buildtime"))
	if err != nil {
		t.Fatal(err)
	}
	take, err := writer.AddFile("canned_opening", "AUDIO", source, 1)
	if err != nil {
		t.Fatal(err)
	}
	if take.SHA256 == "" || take.Path == "" {
		t.Fatalf("incomplete take: %#v", take)
	}
	if err := writer.SetMetadata("canned_opening", 1200, 0, map[string]string{"voice": "dm"}); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "buildtime", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	asset := manifest.Assets["canned_opening"]
	if asset.Selected != 1 || len(asset.Takes) != 1 || asset.DurationMS != 1200 {
		t.Fatalf("unexpected asset: %#v", asset)
	}
	if _, err := os.Stat(filepath.Join(root, "buildtime", asset.Takes[0].Path)); err != nil {
		t.Fatal(err)
	}
}

func TestManifestWriter_SelectionAndValidation(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"", "../bad", "a\\b"} {
		if _, err := writer.AddFile(name, "IMAGE", "missing", 1); err == nil {
			t.Errorf("AddFile(%q) accepted invalid name", name)
		}
	}
	if err := writer.SelectTake("missing", 1); err == nil {
		t.Fatal("SelectTake accepted unknown asset")
	}
	if err := writer.SetMetadata("missing", 0, 0, nil); err == nil {
		t.Fatal("SetMetadata accepted unknown asset")
	}
}

func TestManifestWriter_LoadsAndReplacesTake(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "asset.bin")
	if err := os.WriteFile(source, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	writer, err := NewManifestWriter(filepath.Join(root, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.AddFile("asset", "IMAGE", source, 2); err != nil {
		t.Fatal(err)
	}
	if err := writer.SelectTake("asset", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(); err != nil {
		t.Fatal(err)
	}
	loaded, err := NewManifestWriter(filepath.Join(root, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if err := loaded.SelectTake("asset", 1); err == nil {
		t.Fatal("SelectTake accepted absent take")
	}
	if _, err := loaded.AddFile("asset", "IMAGE", source, 2); err != nil {
		t.Fatal(err)
	}
	if loaded.manifest.Assets["asset"].Selected != 2 {
		t.Fatal("reload did not preserve selected take")
	}
}
