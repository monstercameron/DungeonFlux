package content

import (
	"encoding/json"
	"testing"
)

func TestDefaultWorldBible_Validates(t *testing.T) {
	bible := DefaultWorldBible()
	if err := bible.Validate(); err != nil {
		t.Fatalf("default world bible should validate: %v", err)
	}
	if bible.ID != "drowned_lantern" || len(bible.Keyterms) == 0 {
		t.Fatalf("default world bible is incomplete: %#v", bible)
	}
}

func TestDefaultWorldBible_KeytermsCoverPremiseNouns(t *testing.T) {
	bible := DefaultWorldBible()
	want := []string{"Mother Vell", "the Drowned Lantern", "the bell tower", "the lamplighter"}
	for _, term := range want {
		found := false
		for _, have := range bible.Keyterms {
			if have == term {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("STT keyterms miss premise noun %q: %#v", term, bible.Keyterms)
		}
	}
}

func TestWorldBible_SecretFor(t *testing.T) {
	bible := DefaultWorldBible()
	text, ok := bible.SecretFor("bell_tower_clue", CondClueGranted)
	if !ok || text == "" {
		t.Fatal("granted clue should open the bell tower secret")
	}
	if _, ok := bible.SecretFor("bell_tower_clue", CondEnter); ok {
		t.Fatal("unopened gate must not reveal the secret")
	}
	if _, ok := bible.SecretFor("missing", CondClueGranted); ok {
		t.Fatal("unknown secret must not resolve")
	}
}

func TestWorldBible_LoadRoundTrip(t *testing.T) {
	raw, err := json.Marshal(DefaultWorldBible())
	if err != nil {
		t.Fatalf("marshal default bible: %v", err)
	}
	loaded, err := LoadWorldBible(raw)
	if err != nil {
		t.Fatalf("load bible JSON: %v", err)
	}
	if loaded.ID != "drowned_lantern" || len(loaded.Secrets) != 1 {
		t.Fatalf("round trip lost bible data: %#v", loaded)
	}
}

func TestWorldBibleValidate_RejectsMalformedInputs(t *testing.T) {
	tests := []struct {
		name string
		edit func(*WorldBible)
	}{
		{"identity", func(b *WorldBible) { b.ID = "" }},
		{"setting", func(b *WorldBible) { b.Setting = "" }},
		{"style", func(b *WorldBible) { b.StyleLock = "" }},
		{"keyterms", func(b *WorldBible) { b.Keyterms = nil }},
		{"secrets", func(b *WorldBible) { b.Secrets = nil }},
		{"secret gate", func(b *WorldBible) { b.Secrets[0].GatedBehind = "never" }},
		{"secret text", func(b *WorldBible) { b.Secrets[0].Text = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bible := DefaultWorldBible()
			test.edit(&bible)
			if err := bible.Validate(); err == nil {
				t.Fatal("Validate accepted malformed bible")
			}
		})
	}
}

func TestLoadWorldBible_RejectsBadJSON(t *testing.T) {
	if _, err := LoadWorldBible([]byte("{oops")); err == nil {
		t.Fatal("malformed JSON should be rejected")
	}
	valid, err := json.Marshal(DefaultWorldBible())
	if err != nil {
		t.Fatalf("marshal default bible: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(valid, &raw); err != nil {
		t.Fatalf("remarshal fixture: %v", err)
	}
	raw["id"] = ""
	broken, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal broken bible: %v", err)
	}
	if _, err := LoadWorldBible(broken); err == nil {
		t.Fatal("invalid bible should be rejected on load")
	}
}
