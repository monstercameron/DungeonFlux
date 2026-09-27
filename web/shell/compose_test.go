//go:build !js || !wasm

package main

import (
	"os"
	"strings"
	"testing"
)

func TestStaticIndex_ContainsAppMount(t *testing.T) {
	page, err := os.ReadFile("static/index.html")
	if err != nil {
		t.Fatalf("read static/index.html: %v", err)
	}

	contents := string(page)
	for _, want := range []string{`id="app"`, `id="status"`, "Loading DungeonFlux"} {
		if !strings.Contains(contents, want) {
			t.Errorf("static index is missing %q", want)
		}
	}
}
