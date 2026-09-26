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
	return map[string]string{"box-sizing": "border-box", "max-width": "min(980px, 78vw)", "margin": "0 auto", "padding": "clamp(18px, 2vw, 34px) clamp(24px, 3vw, 48px)", "border-left": "5px solid #d9a441", "border-radius": "0 12px 12px 0", "background": "linear-gradient(90deg, rgba(15, 17, 23, 0.97), rgba(15, 17, 23, 0.75))", "box-shadow": "0 12px 32px rgba(0, 0, 0, 0.38)", "font-family": "Arial, sans-serif", "pointer-events": "none"}
}
