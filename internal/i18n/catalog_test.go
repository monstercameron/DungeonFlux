package i18n

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestCatalog_InlineTables(t *testing.T) {
	catalog := NewCatalog(map[string]map[string]Entry{
		"EN": {
			"turns.left": {Text: "{n} turn left", Other: "{n} turns left"},
			"ui.ok":      {Text: "OK"},
		},
		"es": {
			"turns.left": {Text: "queda {n} turno", Other: "quedan {n} turnos"},
		},
	})
	if got, want := catalog.Locales(), []string{"en", "es"}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Locales() = %q, want %q", got, want)
	}
	if got := catalog.Keys("fr"); len(got) != 2 || got[0] != "turns.left" || got[1] != "ui.ok" {
		t.Fatalf("Keys(fr) = %q, want default table keys", got)
	}
	if !catalog.Has("es", "turns.left") || catalog.Has("es", "ui.ok") {
		t.Fatal("Has(es) wrong for present and missing keys")
	}
	if catalog.Has("fr", "ui.ok") {
		t.Fatal("Has(fr) consulted the fallback table")
	}
	if entry, ok := catalog.Lookup("es", "ui.ok"); !ok || entry.Text != "OK" {
		t.Fatalf("Lookup(es, ui.ok) = %+v %v, want english fallback", entry, ok)
	}
	if _, ok := catalog.Lookup("es", "no.such.key"); ok {
		t.Fatal("Lookup(es, missing) unexpectedly hit")
	}
	if got := catalog.T("es", "turns.left", map[string]string{"n": "3"}, 3, ""); got != "quedan 3 turnos" {
		t.Fatalf("T plural = %q", got)
	}
	if got := catalog.T("es", "no.such.key", nil, 0, "fallback {x}"); got != "fallback {x}" {
		t.Fatalf("T fallback = %q", got)
	}
	if got := catalog.T("es", "no.such.key", nil, 0, ""); got != "no.such.key" {
		t.Fatalf("T bare key = %q", got)
	}
	if got := Format("Hello {missing}", nil); got != "Hello {missing}" {
		t.Fatalf("Format kept = %q", got)
	}
}

func TestPlural_SelectsFormByCount(t *testing.T) {
	catalog := NewCatalog(map[string]map[string]Entry{
		"en": {"turns.left": {Text: "{n} turn left", Other: "{n} turns left"}},
		"es": {"turns.left": {Text: "queda {n} turno", Other: "quedan {n} turnos"}},
	})
	if got := catalog.T("en", "turns.left", map[string]string{"n": "1"}, 1, ""); got != "1 turn left" {
		t.Fatalf("singular = %q", got)
	}
	if got := catalog.T("en", "turns.left", map[string]string{"n": "3"}, 3, ""); got != "3 turns left" {
		t.Fatalf("plural = %q", got)
	}
	if got := catalog.T("es", "turns.left", map[string]string{"n": "3"}, 3, ""); got != "quedan 3 turnos" {
		t.Fatalf("spanish plural = %q", got)
	}
}

func TestPackage_StaysBrowserSafe(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, spec := range file.Imports {
			imp, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(imp, ".") {
				t.Errorf("%s imports %q: catalog must stay stdlib-only for js/wasm", filepath.Base(path), imp)
			}
		}
	}
}
