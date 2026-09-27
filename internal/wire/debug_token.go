package wire

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// debugTokenFile is the file under the data directory that holds a generated
// debug token, matching the human test server supervisor (REPO-016).
const debugTokenFile = "debug.token"

// resolveDebugToken returns the dfctl debug token. It uses DF_DEBUG_TOKEN when
// set; otherwise it generates a random token, writes it to
// <dataDir>/debug.token with owner-only permissions, and reports its path so a
// local run with server.debug=true starts without any environment setup.
func resolveDebugToken(dataDir string, getenv func(string) string) (token, generatedPath string, err error) {
	if token = strings.TrimSpace(getenv("DF_DEBUG_TOKEN")); token != "" {
		return token, "", nil
	}
	if dataDir == "" {
		return "", "", fmt.Errorf("wire: DF_DEBUG_TOKEN is unset and no data dir to write %s", debugTokenFile)
	}
	data := make([]byte, 32)
	if _, err := cryptorand.Read(data); err != nil {
		return "", "", fmt.Errorf("wire: generate debug token: %w", err)
	}
	token = hex.EncodeToString(data)
	path := filepath.Join(dataDir, debugTokenFile)
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return "", "", fmt.Errorf("wire: create data dir for debug token: %w", err)
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return "", "", fmt.Errorf("wire: write debug token: %w", err)
	}
	return token, path, nil
}
