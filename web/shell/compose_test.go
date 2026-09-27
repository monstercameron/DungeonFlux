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
	for _, want := range []string{`id="app"`, `id="status"`, `id="df-splash"`, `class="df-booting"`, "Welcome to the Drowned Lantern", `cache: "no-cache"`} {
		if !strings.Contains(contents, want) {
			t.Errorf("static index is missing %q", want)
		}
	}
	// The bundle fetch must revalidate, not bypass the cache: no-store forced
	// a full 29 MB download on every visit.
	if strings.Contains(contents, `fetch("/app/dungeonflux.wasm", { cache: "no-store" })`) {
		t.Error("the WASM bundle is fetched with no-store")
	}
}
