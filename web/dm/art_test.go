package dm

import "testing"

type mapArt map[string]string

func (m mapArt) ArtURL(name string) string { return m[name] }

func TestArtURL_resolvesThroughInstalledSource(t *testing.T) {
	tests := []struct {
		name   string
		source ArtSource
		art    string
		want   string
	}{
		{name: "nil source resolves empty", source: nil, art: "ui/title_bg", want: ""},
		{name: "known name", source: mapArt{"ui/title_bg": "blob:title"}, art: "ui/title_bg", want: "blob:title"},
		{name: "unknown name", source: mapArt{"ui/title_bg": "blob:title"}, art: "ui/logo_wordmark", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			SetArtSource(tc.source)
			if got := ArtURL(tc.art); got != tc.want {
				t.Fatalf("ArtURL(%q) = %q, want %q", tc.art, got, tc.want)
			}
		})
	}
	SetArtSource(nil)
}
