package dm

import (
	"reflect"
	"strings"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestCliffhangerModelFromView_UsesCaptionAndStillFallback(t *testing.T) {
	SetArtSource(mapArt{"tower.webp": "blob:tower"})
	t.Cleanup(func() { SetArtSource(nil) })
	view := &dungeonfluxv1.DMView{
		BackgroundUrl: "tower.webp",
		Locale:        "es",
		Subtitle:      &dungeonfluxv1.Subtitle{Text: "The bell rings again."},
	}
	model := CliffhangerModelFromView(view)
	if model.Caption != "The bell rings again." || model.Clip.StillURL != "blob:tower" || model.Locale != "es" {
		t.Fatalf("CliffhangerModelFromView() = %+v", model)
	}
	if !CliffhangerReady(model) {
		t.Fatal("caption plus still should make cliffhanger ready")
	}
}

func TestCliffhangerModelFromView_FallsBackToNarrationThenCannedCopy(t *testing.T) {
	narrated := CliffhangerModelFromView(&dungeonfluxv1.DMView{
		Narration: &dungeonfluxv1.Narration{TextSoFar: "The lanterns die."},
	})
	if narrated.Caption != "The lanterns die." {
		t.Fatalf("narration caption = %q", narrated.Caption)
	}
	fallback := CliffhangerModelFromView(&dungeonfluxv1.DMView{})
	if fallback.Caption != defaultCliffhangerCaption {
		t.Fatalf("fallback caption = %q", fallback.Caption)
	}
}

func TestCliffhangerReady_RequiresVisual(t *testing.T) {
	if CliffhangerReady(CliffhangerModel{Caption: "bell"}) {
		t.Fatal("caption without visual should not be ready")
	}
	if CliffhangerReady(CliffhangerModel{Clip: ClipModel{StillURL: "tower.webp"}}) {
		t.Fatal("visual without caption should not be ready")
	}
}

func TestNewEndCardModel_ContainsTerminalCopyAndAttribution(t *testing.T) {
	model := NewEndCardModel()
	if model.Title == "" || model.Subtitle == "" {
		t.Fatalf("end card copy = %+v", model)
	}
	if model.Attribution != SRDAttribution {
		t.Fatal("end card must use the canonical SRD attribution")
	}
}

func TestEndCardReady_RequiresCanonicalAttribution(t *testing.T) {
	base := NewEndCardModel()
	tests := []struct {
		name  string
		model EndCardModel
		want  bool
	}{
		{name: "complete", model: base, want: true},
		{name: "missing title", model: EndCardModel{Subtitle: base.Subtitle, Attribution: base.Attribution}},
		{name: "missing subtitle", model: EndCardModel{Title: base.Title, Attribution: base.Attribution}},
		{name: "altered attribution", model: EndCardModel{Title: base.Title, Subtitle: base.Subtitle, Attribution: strings.TrimSuffix(base.Attribution, ".")}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EndCardReady(test.model); got != test.want {
				t.Fatalf("EndCardReady() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSRDAttribution_ContainsRequiredLicenseDetails(t *testing.T) {
	for _, want := range []string{
		"System Reference Document 5.2.1",
		"Wizards of the Coast LLC",
		"https://www.dndbeyond.com/srd",
		"Creative Commons Attribution 4.0 International License",
		"https://creativecommons.org/licenses/by/4.0/legalcode",
	} {
		if !strings.Contains(SRDAttribution, want) {
			t.Errorf("SRDAttribution does not contain %q", want)
		}
	}
}

func TestEndCardModelFromView_RecapsTheParty(t *testing.T) {
	view := &dungeonfluxv1.DMView{Locale: "es", BuildCards: []*dungeonfluxv1.BuildCard{
		{PlayerNumber: 1, Name: " Mira ", ClassName: "rogue", PortraitUrl: "ui/species_elf"},
		{PlayerNumber: 2, Name: "", ClassName: "bard"},
		{PlayerNumber: 3, Name: "Rook", ClassName: "PALADIN"},
		{PlayerNumber: 4, Name: "Ash", ClassName: "war_cleric"},
		{PlayerNumber: 5, Name: "Bram", ClassName: "fighter"},
		{PlayerNumber: 6, Name: "Cole", ClassName: "wizard"},
	}}
	model := EndCardModelFromView(view)
	want := []EndHero{
		{Name: "Mira", Class: "Rogue", Portrait: "ui/species_elf"},
		{Name: "Rook", Class: "Paladin"},
		{Name: "Ash", Class: "War Cleric"},
		{Name: "Bram", Class: "Fighter"},
	}
	if !reflect.DeepEqual(model.Heroes, want) {
		t.Fatalf("Heroes = %#v, want %#v", model.Heroes, want)
	}
	if model.Locale != "es" || model.Title != "La campana recuerda." || model.Next != T("es", "dm.end_next", nil) {
		t.Fatalf("spanish copy = %+v", model)
	}
	if !EndCardReady(model) {
		t.Fatal("a view-built end card must be ready")
	}
}

func TestEndCardModelFromView_NilViewKeepsEnglishCopyWithoutHeroes(t *testing.T) {
	model := EndCardModelFromView(nil)
	if len(model.Heroes) != 0 || model.Locale != "en" {
		t.Fatalf("nil view model = %+v", model)
	}
	for name, got := range map[string]string{"header": model.Header, "hook": model.Hook, "party": model.Party, "next": model.Next, "rules": model.Rules} {
		if got == "" || strings.HasPrefix(got, "dm.") {
			t.Errorf("%s copy = %q, want catalog text", name, got)
		}
	}
	if model.Attribution != SRDAttribution {
		t.Fatal("end card must carry the canonical SRD attribution")
	}
}

func TestEndHeroPortrait_PrefersWireArtThenClassCrest(t *testing.T) {
	SetArtSource(mapArt{"ui/species_elf": "blob:elf", "ui/class_rogue": "blob:rogue"})
	t.Cleanup(func() { SetArtSource(nil) })
	tests := []struct {
		name string
		hero EndHero
		want string
	}{
		{name: "wire portrait", hero: EndHero{Portrait: "ui/species_elf", Class: "Rogue"}, want: "blob:elf"},
		{name: "blob passes through", hero: EndHero{Portrait: "blob:made"}, want: "blob:made"},
		{name: "crest fallback", hero: EndHero{Portrait: "ui/species_orc", Class: "Rogue"}, want: "blob:rogue"},
		{name: "nothing loaded", hero: EndHero{Class: "Wizard"}, want: ""},
		{name: "no class", hero: EndHero{}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := endHeroPortrait(test.hero); got != test.want {
				t.Fatalf("endHeroPortrait() = %q, want %q", got, test.want)
			}
		})
	}
}
