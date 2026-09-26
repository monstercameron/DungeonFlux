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

// EndCardComponent renders the terminal end card with the canonical SRD
// attribution inside the shared ornate panel language.
func EndCardComponent(model EndCardModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Main(html.Props{Class: "df-dm-end-card", Role: "main", Style: endCardStyle()},
			html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "210px", "right": "210px", "top": "145px", "bottom": "120px", "display": "grid", "place-items": "center"}},
				OrnatePanel("THE TALE CONTINUES",
					html.Div(html.Props{Style: map[string]string{"width": "860px", "max-width": "100%", "padding": "54px 76px 48px", "box-sizing": "border-box", "text-align": "center"}},
						html.P(html.Props{Class: "df-eyebrow", Style: map[string]string{"margin": "0 0 18px", "color": "#d9a441", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "22px", "letter-spacing": ".2em", "font-weight": "700"}}, ui.Text(T(locale, "dm.eyebrow_end", nil))),
						html.H1(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "76px", "font-weight": "500", "line-height": "1.05"}}, ui.Text(model.Title)),
						html.P(html.Props{Style: map[string]string{"margin": "24px 0 34px", "color": "#c8bda8", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "32px", "line-height": "1.2"}}, ui.Text(model.Subtitle)),
						html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": "0 auto 24px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
						html.H2(html.Props{Style: map[string]string{"margin": "0 0 10px", "color": "#e7c27a", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "18px", "letter-spacing": ".12em", "text-transform": "uppercase"}}, ui.Text(EndRules(locale))),
						html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "16px", "line-height": "1.5"}}, ui.Text(model.Attribution)),
					),
				),
			),
		)
	}
}

func endCardStyle() map[string]string {
	style := map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden", "background": "radial-gradient(circle at 50% 35%,#25232a,#171820 48%,#0f1117 100%)", "color": "#efe6d2"}
	if artURL := ArtURL("ui/end_bg"); artURL != "" {
		style["background-image"] = "linear-gradient(180deg,rgba(15,17,23,.22),rgba(15,17,23,.86)),url('" + artURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return style
}
