//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CliffhangerComponent renders the final generated still and a shared speaker
// caption, with a video clip retained when the runtime has one ready.
func CliffhangerComponent(model CliffhangerModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		clip := model.Clip
		if artURL := ArtURL("ui/cliffhanger"); artURL != "" {
			clip.StillURL = artURL
		}
		return html.Section(html.Props{Class: "df-dm-cliffhanger", Role: "status", Aria: map[string]string{"live": "polite", "label": T(locale, "dm.clip_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden", "background": "#0f1117"}},
			clipNode(clip),
			html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none", "background": "linear-gradient(180deg,rgba(8,10,15,.12) 25%,rgba(8,10,15,.9) 100%),radial-gradient(circle at 50% 42%,transparent 25%,rgba(8,10,15,.55) 100%)"}}),
			html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "260px", "right": "260px", "bottom": "72px", "padding": "22px 30px 10px", "pointer-events": "none"}}, SpeakerCaption(CaptionModel{Speaker: T(locale, "dm.speaker_dm", nil), Text: model.Caption})),
		)
	}
}

func clipNode(model ClipModel) *router.Element {
	component := ClipComponent(model)
	return component(router.Attrs{})
}

// EndCardComponent renders the terminal end card: a header band, the
// epilogue, the party recap, the farewell and next step, and the canonical SRD
// attribution as a quiet footer. Layout and type live in dmEndCardCSS.
func EndCardComponent(model EndCardModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		injectEndCardCSS()
		return html.Main(html.Props{Class: "df-dm-end-card", Role: "main", Lang: model.Locale, Style: endCardStyle()},
			html.Div(html.Props{Class: "df-end-shade", Aria: map[string]string{"hidden": "true"}}),
			html.Div(html.Props{Class: "df-end-stage"},
				html.Section(html.Props{Class: "df-ornate-panel df-end-panel", Role: "region", Aria: map[string]string{"labelledby": "df-end-heading"}},
					endCorner("is-tl"), endCorner("is-tr"), endCorner("is-bl"), endCorner("is-br"),
					html.Header(html.Props{Class: "df-end-header"},
						html.H2(html.Props{ID: "df-end-heading", Class: "df-end-header-title"}, ui.Text(model.Header)),
					),
					html.Div(html.Props{Class: "df-end-body"},
						html.H1(html.Props{Class: "df-end-epilogue df-end-reveal"}, ui.Text(model.Title)),
						html.P(html.Props{Class: "df-end-hook df-end-reveal"}, ui.Text(model.Hook)),
						endParty(model),
						html.Div(html.Props{Class: "df-end-divider df-end-reveal", Aria: map[string]string{"hidden": "true"}}),
						html.P(html.Props{Class: "df-end-thanks df-end-reveal"}, ui.Text(model.Subtitle)),
						html.P(html.Props{Class: "df-end-next df-end-reveal"}, ui.Text(model.Next)),
					),
					html.Footer(html.Props{Class: "df-end-footer df-end-reveal"},
						html.H3(html.Props{Class: "df-end-rules"}, ui.Text(model.Rules)),
						html.P(html.Props{Class: "df-end-attribution"}, ui.Text(model.Attribution)),
					),
				),
			),
		)
	}
}

func endCorner(position string) ui.Node {
	return html.Span(html.Props{Class: "df-end-corner " + position, Aria: map[string]string{"hidden": "true"}})
}

// endParty renders the recap row, or nothing when the view carried no heroes.
func endParty(model EndCardModel) ui.Node {
	if len(model.Heroes) == 0 {
		return html.Span(html.Props{Class: "df-end-party-empty", Hidden: true})
	}
	heroes := make([]ui.Node, 0, len(model.Heroes))
	for _, hero := range model.Heroes {
		heroes = append(heroes, endHeroNode(hero))
	}
	return html.Div(html.Props{Class: "df-end-party df-end-reveal", Role: "group", Aria: map[string]string{"label": model.Party}},
		html.P(html.Props{Class: "df-end-party-label"}, ui.Text(model.Party)),
		html.Ul(html.Props{Class: "df-end-heroes"}, heroes...),
	)
}

func endHeroNode(hero EndHero) ui.Node {
	var portrait ui.Node
	if url := endHeroPortrait(hero); url != "" {
		portrait = html.Img(html.Props{Class: "df-end-hero-img", Src: url, Alt: hero.Name})
	} else {
		portrait = html.Span(html.Props{Class: "df-end-hero-glyph", Aria: map[string]string{"hidden": "true"}})
	}
	return html.Li(html.Props{Class: "df-end-hero"},
		html.Span(html.Props{Class: "df-end-hero-portrait"}, portrait),
		html.Span(html.Props{Class: "df-end-hero-name"}, ui.Text(hero.Name)),
		html.Span(html.Props{Class: "df-end-hero-class"}, ui.Text(hero.Class)),
	)
}

// endCardStyle sets only the backdrop art inline; everything else is in
// dmEndCardCSS so a re-render never leaves stale inline keys behind.
func endCardStyle() map[string]string {
	if artURL := ArtURL("ui/end_bg"); artURL != "" {
		return map[string]string{"background-image": "url('" + artURL + "')"}
	}
	return map[string]string{}
}
