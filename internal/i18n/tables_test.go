package i18n

import (
	"reflect"
	"testing"
)

func TestLookup_FallsBackToEnglish(t *testing.T) {
	catalog := Default()
	cases := []struct {
		name, locale, key, want string
	}{
		{"english move", "en", "move.attack", "Attack the drowned thrall"},
		{"spanish move", "es", "move.attack", "Atacar al ahogado"},
		{"unknown locale falls back", "fr", "move.attack", "Attack the drowned thrall"},
		{"empty locale falls back", "", "move.attack", "Attack the drowned thrall"},
		{"region tag settles", "es-MX", "move.attack", "Atacar al ahogado"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := catalog.T(tc.locale, tc.key, nil, 0, ""); got != tc.want {
				t.Fatalf("T(%q, %q) = %q, want %q", tc.locale, tc.key, got, tc.want)
			}
		})
	}
}

func TestFormat_SubstitutesArguments(t *testing.T) {
	catalog := Default()
	got := catalog.T("en", "reason.WAITING_FOR_PLAYER", map[string]string{"name": "Ari"}, 0, "")
	if got != "Waiting for Ari" {
		t.Fatalf("args render = %q", got)
	}
	got = catalog.T("es", "reason.OUT_OF_RANGE", map[string]string{"ft": "30"}, 0, "")
	if got != "Objetivo fuera de alcance (30 pies)" {
		t.Fatalf("spanish args render = %q", got)
	}
	kept := Format("Hello {missing}", nil)
	if kept != "Hello {missing}" {
		t.Fatalf("unknown placeholder = %q", kept)
	}
}

func TestMissingKey_RendersFallback(t *testing.T) {
	catalog := Default()
	if got := catalog.T("es", "no.such.key", nil, 0, "fallback text"); got != "fallback text" {
		t.Fatalf("fallback = %q", got)
	}
	if got := catalog.T("es", "no.such.key", nil, 0, ""); got != "no.such.key" {
		t.Fatalf("bare key = %q", got)
	}
}

func TestCatalog_LocalesAndKeys(t *testing.T) {
	catalog := Default()
	if got, want := catalog.Locales(), []string{"en", "es"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Locales() = %q, want %q", got, want)
	}
	enKeys := catalog.Keys("en")
	if len(enKeys) == 0 {
		t.Fatal("Keys(en) is empty")
	}
	for i := 1; i < len(enKeys); i++ {
		if enKeys[i-1] >= enKeys[i] {
			t.Fatalf("Keys(en) not sorted at %d: %q >= %q", i, enKeys[i-1], enKeys[i])
		}
	}
	if got, want := catalog.Keys("fr"), enKeys; !reflect.DeepEqual(got, want) {
		t.Fatalf("Keys(fr) = %d keys, want %d default keys", len(got), len(want))
	}
}

func TestCatalog_Has(t *testing.T) {
	catalog := Default()
	cases := []struct {
		name, locale, key string
		want              bool
	}{
		{"english hit", "en", "move.attack", true},
		{"spanish hit", "es", "move.attack", true},
		{"missing key", "en", "move.nope", false},
		{"missing locale", "fr", "move.attack", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := catalog.Has(tc.locale, tc.key); got != tc.want {
				t.Fatalf("Has(%q, %q) = %v, want %v", tc.locale, tc.key, got, tc.want)
			}
		})
	}
}

func TestLocales_SupportedSet(t *testing.T) {
	if !Supported("es") || !Supported("en") {
		t.Fatal("en and es must be supported")
	}
	if Supported("fr") {
		t.Fatal("fr must not be supported")
	}
	if got := Settle("fr", "es"); got != "es" {
		t.Fatalf("settle unknown = %q, want es", got)
	}
	if got := Settle("", "es"); got != "es" {
		t.Fatalf("settle empty = %q, want es", got)
	}
	if got := DetectLocale([]string{"fr-FR", "es-MX"}); got != "es" {
		t.Fatalf("detect = %q, want es", got)
	}
	switcher := NewSwitcher("es")
	if got := switcher.T(Default(), "move.attack", nil); got != "Atacar al ahogado" {
		t.Fatalf("switcher t = %q", got)
	}
}

func TestMessage_RendersCatalogTables(t *testing.T) {
	catalog := Default()
	msg := Text("reason.NOT_YOUR_TURN", "Waiting for {holder}").WithArgs(map[string]string{"holder": "seat 1"})
	if got, want := msg.Render(catalog, "es"), "Esperando a seat 1"; got != want {
		t.Fatalf("Render es = %q, want %q", got, want)
	}
	if got, want := msg.Render(catalog, "en"), "Waiting for seat 1"; got != want {
		t.Fatalf("Render en = %q, want %q", got, want)
	}
}
