package wire

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifest_ResolvesSelectedAssetAndCannedOpening(t *testing.T) {
	path := writeManifest(t, `{"version":1,"assets":{"battlefield_tavern_splat":{"kind":"SPLAT","selected":1,"duration_ms":12,"takes":[{"number":1,"sha256":"`+strings.Repeat("a", 64)+`","path":"assets/scene.sog"}]},"canned_opening":{"kind":"AUDIO","selected":1,"contact_ms":345,"takes":[{"number":1,"sha256":"`+strings.Repeat("b", 64)+`","path":"assets/opening.wav"}]}}}`)
	result, err := LoadManifest(path, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if result.Hash == "" {
		t.Fatal("LoadManifest() returned empty digest")
	}
	for _, asset := range result.OneShot.Catalogue {
		if asset.ID == "battlefield_tavern_splat" && asset.URL != "/assets/scene.sog" {
			t.Fatalf("scene URL = %q", asset.URL)
		}
		if asset.ID == "canned_opening" && asset.Meta["contact_ms"] != "345" {
			t.Fatalf("opening contact metadata = %#v", asset.Meta)
		}
	}
}

func TestLoadManifest_MissingFallsBack(t *testing.T) {
	result, err := LoadManifest(filepath.Join(t.TempDir(), "missing.json"), slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	if result.Hash != "" || result.OneShot.ID == "" {
		t.Fatalf("missing manifest result = %#v", result)
	}
}

func TestLoadManifest_InvalidAndDigest(t *testing.T) {
	path := writeManifest(t, `{"version":1,"assets":{}}`)
	result, err := LoadManifest(path, nil)
	if err != nil {
		t.Fatalf("LoadManifest() error = %v", err)
	}
	data, _ := os.ReadFile(path)
	want := sha256.Sum256(data)
	if result.Hash != hex.EncodeToString(want[:]) {
		t.Fatalf("digest = %q, want %x", result.Hash, want)
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadManifest(bad, nil); err == nil {
		t.Fatal("LoadManifest() accepted malformed JSON")
	}
}

func writeManifest(t *testing.T, data string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
