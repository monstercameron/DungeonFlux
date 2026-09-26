package wire

import (
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"rsc.io/qr"
)

// writeJoinQR writes the phone join URL into the configured runtime asset
// directory and returns its content-addressed public asset URL.
func writeJoinQR(dataDir, joinURL string) (string, error) {
	if dataDir == "" || joinURL == "" {
		return "", fmt.Errorf("wire: QR data directory and join URL are required")
	}
	parsed, err := url.Parse(joinURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("wire: invalid QR join URL")
	}
	code, err := qr.Encode(joinURL, qr.M)
	if err != nil {
		return "", fmt.Errorf("wire: encode join QR: %w", err)
	}
	data := code.PNG()
	digest := sha256.Sum256(data)
	name := fmt.Sprintf("%x.png", digest)
	dir := filepath.Join(dataDir, "assets")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("wire: create QR asset directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		return "", fmt.Errorf("wire: write join QR: %w", err)
	}
	return "/assets/" + name, nil
}
