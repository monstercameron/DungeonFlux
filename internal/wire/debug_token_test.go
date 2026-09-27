package wire

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveDebugToken(t *testing.T) {
	tests := []struct {
		name          string
		env           string
		dataDir       func(t *testing.T) string
		wantToken     string
		wantGenerated bool
		wantErr       bool
	}{
		{name: "uses the environment token", env: " env-token ", dataDir: func(t *testing.T) string { return t.TempDir() }, wantToken: "env-token"},
		{name: "generates and writes a token when unset", dataDir: func(t *testing.T) string { return filepath.Join(t.TempDir(), "runtime") }, wantGenerated: true},
		{name: "fails when unset without a data dir", dataDir: func(*testing.T) string { return "" }, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dataDir := tc.dataDir(t)
			getenv := func(key string) string {
				if key == "DF_DEBUG_TOKEN" {
					return tc.env
				}
				return ""
			}
			token, path, err := resolveDebugToken(dataDir, getenv)
			if (err != nil) != tc.wantErr {
				t.Fatalf("resolveDebugToken() error = %v, wantErr %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
			if !tc.wantGenerated {
				if token != tc.wantToken || path != "" {
					t.Fatalf("resolveDebugToken() = %q, %q; want %q and no file", token, path, tc.wantToken)
				}
				if _, statErr := os.Stat(filepath.Join(dataDir, debugTokenFile)); !os.IsNotExist(statErr) {
					t.Fatalf("token file written although DF_DEBUG_TOKEN was set: %v", statErr)
				}
				return
			}
			if len(token) != 64 || path != filepath.Join(dataDir, debugTokenFile) {
				t.Fatalf("generated token %q at %q", token, path)
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || strings.TrimSpace(string(data)) != token {
				t.Fatalf("token file = %q, %v; want %q", data, readErr, token)
			}
			// Windows has no Unix permission bits (files report 0666), so the
			// owner-only mode is only checked where it exists.
			if info, _ := os.Stat(path); runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
				t.Fatalf("token file mode = %v, want 0600", info.Mode().Perm())
			}
		})
	}
}
