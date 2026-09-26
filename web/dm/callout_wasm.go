//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CalloutComponent renders a short DM steering message over the scene.
func CalloutComponent(view CalloutView) router.Component {
	return func(_ router.Attrs) *router.Element {
		return html.Div(html.Props{Class: "df-dm-callout", Hidden: !view.Visible, Role: "status", Aria: map[string]string{"live": "polite"}, Style: calloutStyle()},
			html.Span(html.Props{Class: "df-dm-callout-label", Style: map[string]string{"display": "block", "margin-bottom": "8px", "color": "#d9a441", "font-size": "clamp(16px, 1.5vw, 26px)", "font-weight": "700", "letter-spacing": "0.12em", "text-transform": "uppercase"}}, ui.Text("DM steering")),
			html.P(html.Props{Class: "df-dm-callout-copy", Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(24px, 2.5vw, 48px)", "line-height": "1.12"}}, ui.Text(view.Text)),
		)
	}
}

func calloutStyle() map[string]string {
	style := map[string]string{"box-sizing": "border-box", "max-width": "min(1280px, 82vw)", "margin": "0 auto", "padding": "clamp(28px, 3vw, 58px) clamp(42px, 5vw, 92px)", "border": "1px solid rgba(217, 164, 65, 0.72)", "border-radius": "12px", "background": "linear-gradient(90deg, rgba(15, 17, 23, 0.97), rgba(15, 17, 23, 0.75))", "box-shadow": "0 12px 32px rgba(0, 0, 0, 0.38)", "font-family": "Arial, sans-serif", "pointer-events": "none"}
	if artURL := ArtURL("ui/banner_callout"); artURL != "" {
		style["background-image"] = "linear-gradient(90deg, rgba(15,17,23,.18), rgba(15,17,23,.18)), url('" + artURL + "')"
		style["background-size"] = "100% 100%"
		style["background-position"] = "center"
	}
	return style
}
