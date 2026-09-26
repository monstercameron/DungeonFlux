package wire

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteJoinQR_WritesPNGAsset(t *testing.T) {
	dir := t.TempDir()
	url, err := writeJoinQR(dir, 18111, "DF-ROOM")
	if err != nil {
		t.Fatalf("writeJoinQR() error = %v", err)
	}
	if url != "/assets/join-room.png" {
		t.Fatalf("URL = %q", url)
	}
	data, err := os.ReadFile(filepath.Join(dir, "assets", "join-room.png"))
	if err != nil || len(data) < 8 {
		t.Fatalf("QR asset missing: %v", err)
	}
}

func TestWriteJoinQR_PNGHeader(t *testing.T) {
	dir := t.TempDir()
	if _, err := writeJoinQR(dir, 18111, "DF ROOM"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "assets", joinQRName))
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
