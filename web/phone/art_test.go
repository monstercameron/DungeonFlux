package phone

import "testing"

type phoneArtFake struct{}

func (phoneArtFake) ArtURL(name string) string { return " /assets/" + name + ".webp " }

func TestArtSource_ResolvesAndTrimsManifestURLs(t *testing.T) {
	SetArtSource(phoneArtFake{})
	t.Cleanup(func() { SetArtSource(nil) })
	if got := ArtURL("ui/phone_bg"); got != "/assets/ui/phone_bg.webp" {
		t.Fatalf("ArtURL did not trim source URL: %q", got)
	}
}

func TestArtAssets_MapPhoneControls(t *testing.T) {
	for _, test := range []struct {
		name, got, want string
	}{
		{"attack icon", moveArtAsset(" attack "), "ui/icon_attack"},
		{"conversation icon", moveArtAsset("talk_vell"), "ui/icon_talk"},
		{"roll icon", moveArtAsset("roll_hero"), "ui/icon_ready"},
		{"unknown icon", moveArtAsset("missing"), ""},
		{"species", speciesArtAsset(" Elf "), "ui/species_elf"},
		{"empty species", speciesArtAsset(" "), ""},
		{"class crest", classArtAsset(" Rogue "), "ui/class_rogue"},
		{"unsupported crest", classArtAsset("Fighter"), ""},
		{"bloodied", statusArtAsset("BLOODIED"), "ui/status_bloodied"},
		{"down", statusArtAsset("You're down"), "ui/status_down"},
		{"focused", statusArtAsset("Focused"), "ui/status_spotlight"},
		{"unknown status", statusArtAsset("Inspired"), ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestArtHelpers_ProvideFallbackStyles(t *testing.T) {
	SetArtSource(nil)
	if got := artBackground("", "fallback"); got != "fallback" {
		t.Fatalf("fallback background = %q", got)
	}
	style := artButtonStyle(buttonPrimaryAsset, "#d9a441")
	if style["background-color"] != "#d9a441" || style["background-image"] != "" {
		t.Fatalf("fallback button style = %#v", style)
	}
	SetArtSource(phoneArtFake{})
	if got := artBackground("/asset.webp", "fallback"); got != `url("/asset.webp")` {
		t.Fatalf("image background = %q", got)
	}
	if style := artButtonStyle(buttonPrimaryAsset, "#d9a441"); style["background-image"] == "" {
		t.Fatal("loaded button style has no image")
	}
}

func TestPortraitSrc_ResolvesAssetSelectorsAndPassesBrowserURLs(t *testing.T) {
	SetArtSource(phoneArtFake{})
	t.Cleanup(func() { SetArtSource(nil) })
	for _, test := range []struct {
		name, input, want string
	}{
		{name: "logical asset", input: "mother_vell", want: "/assets/mother_vell.webp"},
		{name: "blob URL", input: " blob:ready ", want: "blob:ready"},
		{name: "data URL", input: "data:image/webp;base64,abc", want: "data:image/webp;base64,abc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := portraitSrc(test.input); got != test.want {
				t.Fatalf("portraitSrc(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestHeroProxyArt_UsesSpeciesBeforeClass(t *testing.T) {
	SetArtSource(phoneArtFake{})
	t.Cleanup(func() { SetArtSource(nil) })
	if got := heroProxyArt("elf", "rogue", "seed"); got != "/assets/ui/species_elf.webp" {
		t.Fatalf("heroProxyArt() = %q, want species proxy", got)
	}
}
