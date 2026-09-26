package i18n

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestCatalogs_HaveParityAcrossLocales fails when a locale is missing keys
// or carries keys no other locale has.
func TestCatalogs_HaveParityAcrossLocales(t *testing.T) {
	english := EnglishEntries()
	spanish := SpanishEntries()
	for key := range english {
		if _, ok := spanish[key]; !ok {
			t.Errorf("spanish catalog misses key %q", key)
		}
	}
	for key := range spanish {
		if _, ok := english[key]; !ok {
			t.Errorf("spanish catalog has extra key %q", key)
		}
	}
}

// TestCatalogs_CoverEveryScreenKey fails when a screen key has no entry in a
// committed catalog.
func TestCatalogs_CoverEveryScreenKey(t *testing.T) {
	catalog := Default()
	for screen, keys := range ScreenKeys() {
		for _, key := range keys {
			if !catalog.Has("en", key) {
				t.Errorf("screen %s key %q misses english entry", screen, key)
			}
			if !catalog.Has("es", key) {
				t.Errorf("screen %s key %q misses spanish entry", screen, key)
			}
		}
	}
}

// TestCatalogs_HaveNoUnusedKeys fails when a catalog key is attributed to no
// screen, so dead copy cannot accumulate unnoticed.
func TestCatalogs_HaveNoUnusedKeys(t *testing.T) {
	used := make(map[string]string)
	for screen, keys := range ScreenKeys() {
		for _, key := range keys {
			used[key] = screen
		}
	}
	keys := RequiredKeys()
	sort.Strings(keys)
	for _, key := range keys {
		if _, ok := used[key]; !ok {
			t.Errorf("key %q is in no screen key list", key)
		}
	}
}

var hardTextPattern = regexp.MustCompile("(html|ui)[.]Text[(][\"]([^\"]*)[\"]" + "[)]")

var hardTextAttrPattern = regexp.MustCompile("(Placeholder|Alt):[ ]*[\"]([^\"]*)[\"]")

// TestScreens_HaveNoHardCodedText fails when a client view renders a
// hard-coded literal instead of going through the catalog. Dynamic values,
// empty attributes, and test expectations are not screen copy.
func TestScreens_HaveNoHardCodedText(t *testing.T) {
	root := repoRoot(t)
	// E2E-008 owns the DM and phone surfaces. Host and shell copy is gated
	// by the workers that own those packages, so do not cross lane boundaries.
	screens := []string{"phone", "dm"}
	for _, screen := range screens {
		dir := filepath.Join(root, "web", screen)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			for _, found := range hardTextPattern.FindAllSubmatch(data, -1) {
				if len(found) > 2 && len(found[2]) > 0 {
					t.Errorf("web/%s/%s renders hard-coded text %q: add a catalog key and render through t()", screen, name, found[2])
				}
			}
			for _, found := range hardTextAttrPattern.FindAllSubmatch(data, -1) {
				if len(found[2]) > 0 {
					t.Errorf("web/%s/%s renders hard-coded attribute %q: add a catalog key and render through t()", screen, name, found[2])
				}
			}
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found above the i18n package")
		}
		dir = parent
	}
}
