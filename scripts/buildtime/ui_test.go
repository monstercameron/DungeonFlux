package main

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestCheckUI_NormalizesNamesAndSortsMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "present.webp"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing, err := CheckUI(root, []string{"ui/zulu", "present", "ui/alpha"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"ui/alpha", "ui/zulu"}
	if !reflect.DeepEqual(missing, want) {
		t.Fatalf("missing = %v, want %v", missing, want)
	}
}

func TestRegisterUIFile_UsesUILogicalName(t *testing.T) {
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "hero.webp")
	if err := os.WriteFile(source, []byte("webp"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := registerUIFile(writer, source); err != nil {
		t.Fatal(err)
	}
	if _, ok := writer.manifest.Assets["ui/hero"]; !ok {
		t.Fatalf("manifest assets = %#v", writer.manifest.Assets)
	}
	if _, ok := writer.manifest.Assets["hero"]; ok {
		t.Fatal("temporary non-UI logical name remained")
	}
}

func TestTrimUIPNG_TrimsUniformThemeMargins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "button_primary.png")
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	for y := 0; y < 5; y++ {
		for x := 0; x < 5; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 0x0f, G: 0x11, B: 0x17, A: 0xff})
		}
	}
	img.SetRGBA(2, 2, color.RGBA{R: 0xff, G: 0, B: 0, A: 0xff})
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	trimmed, err := trimUIPNG(path)
	if err != nil {
		t.Fatal(err)
	}
	if trimmed == path {
		t.Fatal("expected a trimmed temporary image")
	}
	width, height, err := imageSize(trimmed)
	if err != nil {
		t.Fatal(err)
	}
	if width != 1 || height != 1 {
		t.Fatalf("trimmed size = %dx%d, want 1x1", width, height)
	}
	_ = os.Remove(trimmed)
}

func TestCappedSize_PreservesAspectRatio(t *testing.T) {
	if width, height := cappedSize(4000, 2000, 1920); width != 1920 || height != 960 {
		t.Fatalf("wide size = %dx%d", width, height)
	}
	if width, height := cappedSize(1000, 800, 1920); width != 1000 || height != 800 {
		t.Fatalf("small size = %dx%d", width, height)
	}
}

func TestConvertUI_StopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "ui"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ui", "one.png"), []byte("not a PNG"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ConvertUI(ctx, root); err == nil {
		t.Fatal("ConvertUI accepted cancelled context")
	}
}
