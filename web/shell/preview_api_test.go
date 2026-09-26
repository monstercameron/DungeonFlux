package main

import "testing"

type previewRegistryFixture struct{ fixtures []PreviewFixture }

func (r previewRegistryFixture) PreviewFixtures() []PreviewFixture { return r.fixtures }

func TestPreviewCatalog_FixturesQualifiesSortsAndDropsEmpty(t *testing.T) {
	catalog := NewPreviewCatalog(
		previewRegistryFixture{fixtures: []PreviewFixture{{Name: "end", Label: "End"}, {Name: "", Label: "ignored"}, {Name: "lobby", Label: ""}, {Name: "lobby", Label: "duplicate"}}},
		previewRegistryFixture{fixtures: []PreviewFixture{{Name: "combat", Label: "Combat"}}},
	)
	got := catalog.Fixtures()
	if len(got) != 3 {
		t.Fatalf("got %d fixtures, want 3: %#v", len(got), got)
	}
	want := []PreviewFixture{{Name: "dm:end", Label: "End"}, {Name: "dm:lobby", Label: "lobby"}, {Name: "p:combat", Label: "Combat"}}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("fixture %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestPreviewCatalog_NilRegistriesAreSafe(t *testing.T) {
	if got := (PreviewCatalog{}).Fixtures(); got != nil {
		t.Fatalf("nil catalog fixtures = %#v, want nil", got)
	}
}

func TestPreviewNameFromSearch_ParsesBrowserSearch(t *testing.T) {
	tests := []struct {
		name   string
		search string
		want   string
	}{
		{name: "browser leading question mark", search: "?preview=combat-my-turn", want: "combat-my-turn"},
		{name: "query without question mark", search: "preview=creation-pick", want: "creation-pick"},
		{name: "escaped fixture name", search: "?preview=typed-input%20ready", want: "typed-input ready"},
		{name: "missing preview", search: "?room=demo", want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := previewNameFromSearch(test.search); got != test.want {
				t.Fatalf("previewNameFromSearch(%q) = %q, want %q", test.search, got, test.want)
			}
		})
	}
}
