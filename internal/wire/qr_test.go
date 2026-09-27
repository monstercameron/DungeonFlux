package wire

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJoinQR_WritesPNGAsset(t *testing.T) {
	dir := t.TempDir()
	url, err := writeJoinQR(dir, "http://192.168.1.2:18111/p?room=DF-ROOM")
	if err != nil {
		t.Fatalf("writeJoinQR() error = %v", err)
	}
	if len(url) != len("/assets/")+64+4 {
		t.Fatalf("URL = %q, want content-addressed PNG", url)
	}
	data, err := os.ReadFile(filepath.Join(dir, "assets", filepath.Base(url)))
	if err != nil || len(data) < 8 {
		t.Fatalf("QR asset missing: %v", err)
	}
}

func TestWriteJoinQR_PNGHeader(t *testing.T) {
	dir := t.TempDir()
	path, err := writeJoinQR(dir, "http://localhost:18111/p?room=DF+ROOM")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "assets", filepath.Base(path)))
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	for i, value := range want {
		if len(data) <= i || data[i] != value {
			t.Fatalf("PNG signature mismatch at byte %d", i)
		}
	}
}

func TestWriteJoinQR_AssetResolves(t *testing.T) {
	dir := t.TempDir()
	path, err := writeJoinQR(dir, "http://192.168.1.2:18111/p?room=DF-ROOM")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	assetHandler(filepath.Join(dir, "assets")).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("QR GET status = %d, want 200", response.Code)
	}
	digest := sha256.Sum256(response.Body.Bytes())
	if filepath.Base(path) != fmt.Sprintf("%x.png", digest) {
		t.Fatalf("QR path %q does not match content hash", path)
	}
}

func TestBuild_QRAssetResolvesThroughHandler(t *testing.T) {
	cfg := testConfig(t)
	app, err := Build(context.Background(), cfg, []byte("qr-test"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = app.Close() }()
	entries, err := os.ReadDir(filepath.Join(cfg.Server.DataDir, "assets"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".png" {
			continue
		}
		request := httptest.NewRequest(http.MethodGet, "/assets/"+entry.Name(), nil)
		response := httptest.NewRecorder()
		app.Handler().ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("QR GET status = %d, want 200", response.Code)
		}
		return
	}
	t.Fatal("Build did not create a QR PNG")
}
