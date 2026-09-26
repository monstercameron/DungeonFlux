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

func TestArtSrc_PassesBrowserURLsAndResolvesAssetSelectors(t *testing.T) {
	SetArtSource(mapArt{
		"mother_vell": "blob:vell",
		"e9d36f1b11ccbed4c55847d2358a913051f8214b37d96a9e49783ce702515e7d":             "blob:vell",
		"/assets/e9d36f1b11ccbed4c55847d2358a913051f8214b37d96a9e49783ce702515e7d.png": "blob:vell",
	})
	t.Cleanup(func() { SetArtSource(nil) })
	tests := []struct {
		name string
		want string
	}{
		{name: "blob URL", want: "blob:already"},
		{name: "data URL", want: "data:image/webp;base64,abc"},
		{name: "logical name", want: "blob:vell"},
		{name: "sha selector", want: "blob:vell"},
		{name: "/assets SHA path", want: "blob:vell"},
		{name: "/assets path", want: ""},
		{name: "unknown selector", want: ""},
		{name: "blank selector", want: ""},
	}
	selectors := []string{" blob:already ", "data:image/webp;base64,abc", "mother_vell", "e9d36f1b11ccbed4c55847d2358a913051f8214b37d96a9e49783ce702515e7d", "/assets/e9d36f1b11ccbed4c55847d2358a913051f8214b37d96a9e49783ce702515e7d.png", "/assets/preview/missing.webp", "missing", "  "}
	for index, selector := range selectors {
		t.Run(tests[index].name, func(t *testing.T) {
			if got := artSrc(selector); got != tests[index].want {
				t.Fatalf("artSrc(%q) = %q, want %q", selector, got, tests[index].want)
			}
		})
	}
}
