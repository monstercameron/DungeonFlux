//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CliffhangerComponent renders the final cinematic with a readable lower-third.
func CliffhangerComponent(model CliffhangerModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Section(html.Props{Class: "df-dm-cliffhanger", Role: "status", Aria: map[string]string{"live": "polite", "label": T(locale, "dm.clip_label", nil)}, Style: map[string]string{
			"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden", "background": "#0f1117",
		}},
			clipNode(model.Clip),
			html.Div(html.Props{Class: "df-dm-cliffhanger-vignette", Style: map[string]string{
				"position": "absolute", "inset": "0", "pointer-events": "none", "background": "linear-gradient(180deg, rgba(15,17,23,.08) 28%, rgba(15,17,23,.88) 100%)",
			}}),
			html.Div(html.Props{Class: "df-dm-cliffhanger-caption", Style: map[string]string{
				"position": "absolute", "left": "6%", "right": "6%", "bottom": "7%", "padding": "1.25rem 1.5rem", "box-sizing": "border-box", "border-left": "4px solid #d9a441", "background": "rgba(15,17,23,.84)", "box-shadow": "0 8px 30px rgba(0,0,0,.35)", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1.25rem, 2.4vw, 2.5rem)", "line-height": "1.3",
			}}, html.P(html.Props{Style: map[string]string{"margin": "0"}}, ui.Text(model.Caption))),
		)
	}
}

func clipNode(model ClipModel) *router.Element {
	component := ClipComponent(model)
	return component(router.Attrs{})
}

// EndCardComponent renders the terminal end card for the DM screen.
func EndCardComponent(model EndCardModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Main(html.Props{Class: "df-dm-end-card", Role: "main", Style: endCardStyle()},
			html.Div(html.Props{Class: "df-dm-end-card-content", Style: map[string]string{"width": "min(88%, 1100px)", "padding": "clamp(1.5rem, 4vw, 4rem)", "box-sizing": "border-box", "border": "1px solid rgba(217,164,65,.7)", "border-radius": "14px", "background": "rgba(15,17,23,.76)", "box-shadow": "inset 0 0 0 1px rgba(239,230,210,.08), 0 16px 50px rgba(0,0,0,.35)"}},
				html.P(html.Props{Class: "df-eyebrow", Style: map[string]string{"margin": "0 0 1rem", "color": "#d9a441", "font-family": "Arial, sans-serif", "font-size": "clamp(.8rem, 1.1vw, 1.2rem)", "letter-spacing": ".2em", "font-weight": "700"}}, ui.Text(T(locale, "dm.eyebrow_end", nil))),
				html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(2.5rem, 6vw, 6rem)", "line-height": "1.05", "font-weight": "500"}}, ui.Text(model.Title)),
				html.P(html.Props{Class: "df-dm-end-card-subtitle", Style: map[string]string{"margin": "1.25rem 0 2.5rem", "color": "#a89f8c", "font-family": "Arial, sans-serif", "font-size": "clamp(1rem, 2vw, 1.8rem)"}}, ui.Text(model.Subtitle)),
				html.Section(html.Props{Class: "df-dm-end-card-attribution", Aria: map[string]string{"label": "SRD attribution"}, Style: map[string]string{"margin": "0 auto", "max-width": "900px", "padding-top": "1.25rem", "border-top": "1px solid rgba(217,164,65,.35)", "text-align": "left"}},
					html.H2(html.Props{Style: map[string]string{"margin": "0 0 .65rem", "color": "#d9a441", "font-family": "Arial, sans-serif", "font-size": "clamp(.85rem, 1.2vw, 1.15rem)", "letter-spacing": ".08em", "text-transform": "uppercase"}}, ui.Text(EndRules(locale))),
					html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-family": "Arial, sans-serif", "font-size": "clamp(.7rem, 1vw, 1rem)", "line-height": "1.55"}}, ui.Text(model.Attribution)),
				),
			),
		)
	}
}

func endCardStyle() map[string]string {
	style := map[string]string{"width": "100%", "height": "100%", "display": "grid", "place-items": "center", "padding": "6%", "box-sizing": "border-box", "background": "radial-gradient(circle at 50% 35%, #25232a 0%, #171820 48%, #0f1117 100%)", "color": "#efe6d2", "text-align": "center"}
	if artURL := ArtURL("ui/end_bg"); artURL != "" {
		style["background-image"] = "linear-gradient(180deg, rgba(15,17,23,.18), rgba(15,17,23,.86)), url('" + artURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return style
}
