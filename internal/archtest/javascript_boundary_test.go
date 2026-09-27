package archtest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJavaScriptBoundary(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		allowed bool
	}{
		{"game renderer", "web/splat/js/scene.mjs", true},
		{"documentation", "docs/search.js", true},
		{"standalone marketing", "website/script.js", true},
		{"nested marketing", "website/components/menu.mjs", true},
		{"phone remains Go", "web/phone/ui.js", false},
		{"DM remains Go", "web/dm/ui.mjs", false},
		{"renderer prefix lookalike", "web/splat-other/ui.js", false},
		{"marketing prefix lookalike", "website-other/script.js", false},
		{"root script", "script.js", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, filepath.FromSlash(tc.path))
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("// fixture\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			got := checkJavaScript(root)
			if (len(got) == 0) != tc.allowed {
				t.Fatalf("checkJavaScript(%q) = %v; allowed = %v", tc.path, got, tc.allowed)
			}
		})
	}
}
