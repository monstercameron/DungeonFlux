package wire

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"rsc.io/qr"
)

const joinQRName = "join-room.png"

// writeJoinQR writes the phone join URL into the configured runtime asset
// directory and returns its public asset URL.
func writeJoinQR(dataDir string, port int, room string) (string, error) {
	if dataDir == "" || room == "" {
		return "", fmt.Errorf("wire: QR data directory and room are required")
	}
	joinURL := fmt.Sprintf("http://localhost:%d/p?room=%s", port, url.QueryEscape(room))
	code, err := qr.Encode(joinURL, qr.M)
	if err != nil {
		return "", fmt.Errorf("wire: encode join QR: %w", err)
	}
	dir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("wire: create QR asset directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, joinQRName), code.PNG(), 0o600); err != nil {
		return "", fmt.Errorf("wire: write join QR: %w", err)
	}
	return "/assets/" + joinQRName, nil
}
