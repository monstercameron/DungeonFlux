package dm

import dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

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

// EndCardModel contains the fixed copy shown after the cliffhanger line.
type EndCardModel struct {
	Title       string
	Subtitle    string
	Attribution string
	Locale      string
}

// SRDAttribution is the required SRD 5.2.1 attribution statement.
const SRDAttribution = `This work includes material from the System Reference Document 5.2.1 ("SRD 5.2.1") by Wizards of the Coast LLC, available at https://www.dndbeyond.com/srd. The SRD 5.2.1 is licensed under the Creative Commons Attribution 4.0 International License, available at https://creativecommons.org/licenses/by/4.0/legalcode.`

// NewEndCardModel creates the end-card copy for the terminal phase state.
func NewEndCardModel() EndCardModel {
	return EndCardModel{
		Title:       T("en", "dm.end_title", nil),
		Subtitle:    T("en", "dm.end_subtitle", nil),
		Attribution: SRDAttribution,
	}
}

// Localized returns the end-card copy rendered for a locale.
func (m EndCardModel) Localized(locale string) EndCardModel {
	locale = localeOrDefault(locale)
	m.Locale = locale
	m.Title = T(locale, "dm.end_title", nil)
	m.Subtitle = T(locale, "dm.end_subtitle", nil)
	return m
}

// EndCardReady reports whether the model has all copy needed to render.
func EndCardReady(model EndCardModel) bool {
	return model.Title != "" && model.Subtitle != "" && model.Attribution == SRDAttribution
}
