package dm

import (
	"strings"
	"unicode"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

const defaultCliffhangerCaption = "Midnight. The tower bell tolls, and every lantern gutters out. Whoever pulls that rope already knows your names."

// CliffhangerModel contains the final still, caption, and playback state.
type CliffhangerModel struct {
	Clip    ClipModel
	Caption string
	Locale  string
}

// CliffhangerModelFromView projects a terminal DM snapshot into a render model.
func CliffhangerModelFromView(view *dungeonfluxv1.DMView) CliffhangerModel {
	if view == nil {
		return CliffhangerModel{Caption: defaultCliffhangerCaption}
	}
	caption := view.GetSubtitle().GetText()
	if caption == "" {
		caption = view.GetNarration().GetTextSoFar()
	}
	if caption == "" {
		caption = defaultCliffhangerCaption
	}
	return CliffhangerModel{Clip: ClipModelFromView(view), Caption: caption, Locale: localeOrDefault(view.GetLocale())}
}

// CliffhangerReady reports whether the terminal caption and a visual fallback exist.
func CliffhangerReady(model CliffhangerModel) bool {
	return model.Caption != "" && (model.Clip.VideoURL != "" || model.Clip.StillURL != "")
}

// EndCardModel contains the copy and party recap shown after the cliffhanger.
// Title is the epilogue line, Subtitle the thank-you line; Header, Hook, Party,
// Next and Rules are the supporting copy, and Heroes recaps who played.
type EndCardModel struct {
	Header      string
	Title       string
	Hook        string
	Party       string
	Subtitle    string
	Next        string
	Rules       string
	Attribution string
	Locale      string
	Heroes      []EndHero
}

// EndHero is one hero in the end-card recap: a name, a display class, and the
// wire portrait selector (resolved through the art source at render time).
type EndHero struct {
	Name     string
	Class    string
	Portrait string
}

// maxEndHeroes caps the recap row; the demo table seats two to four players.
const maxEndHeroes = 4

// SRDAttribution is the required SRD 5.2.1 attribution statement.
const SRDAttribution = `This work includes material from the System Reference Document 5.2.1 ("SRD 5.2.1") by Wizards of the Coast LLC, available at https://www.dndbeyond.com/srd. The SRD 5.2.1 is licensed under the Creative Commons Attribution 4.0 International License, available at https://creativecommons.org/licenses/by/4.0/legalcode.`

// NewEndCardModel creates the English end-card copy for the terminal phase.
func NewEndCardModel() EndCardModel {
	return EndCardModel{Attribution: SRDAttribution}.Localized("en")
}

// EndCardModelFromView builds the localized end card for a terminal DM
// snapshot, recapping the party from its build cards when the view has them.
func EndCardModelFromView(view *dungeonfluxv1.DMView) EndCardModel {
	model := NewEndCardModel().Localized(view.GetLocale())
	for _, card := range view.GetBuildCards() {
		if len(model.Heroes) == maxEndHeroes {
			break
		}
		name := strings.TrimSpace(card.GetName())
		if name == "" {
			continue
		}
		model.Heroes = append(model.Heroes, EndHero{Name: name, Class: endHeroClass(card.GetClassName()), Portrait: strings.TrimSpace(card.GetPortraitUrl())})
	}
	return model
}

// Localized returns the end-card copy rendered for a locale.
func (m EndCardModel) Localized(locale string) EndCardModel {
	locale = localeOrDefault(locale)
	m.Locale = locale
	m.Header = T(locale, "dm.end_header", nil)
	m.Title = T(locale, "dm.end_title", nil)
	m.Hook = T(locale, "dm.end_hook", nil)
	m.Party = T(locale, "dm.end_party", nil)
	m.Subtitle = T(locale, "dm.end_subtitle", nil)
	m.Next = T(locale, "dm.end_next", nil)
	m.Rules = T(locale, "dm.end_rules", nil)
	return m
}

// EndCardReady reports whether the model has all copy needed to render.
func EndCardReady(model EndCardModel) bool {
	return model.Title != "" && model.Subtitle != "" && model.Attribution == SRDAttribution
}

// endHeroClass turns a wire class id ("paladin", "war_cleric") into display
// case ("Paladin", "War Cleric").
func endHeroClass(class string) string {
	words := strings.Fields(strings.ReplaceAll(strings.TrimSpace(class), "_", " "))
	for index, word := range words {
		runes := []rune(strings.ToLower(word))
		runes[0] = unicode.ToUpper(runes[0])
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

// endHeroPortrait resolves a hero's portrait: the wire selector first, then
// the class crest; "" while neither has loaded, so the medallion shows a glyph.
func endHeroPortrait(hero EndHero) string {
	if url := artSrc(hero.Portrait); url != "" {
		return url
	}
	if hero.Class == "" {
		return ""
	}
	return ArtURL(classCrestArt(hero.Class))
}
