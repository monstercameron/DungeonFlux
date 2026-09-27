//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CalloutComponent renders the DM's steering beat as a centered ornate banner.
func CalloutComponent(view CalloutView) router.Component {
	return func(_ router.Attrs) *router.Element {
		return html.Div(html.Props{Class: "df-dm-callout", Hidden: !view.Visible, Role: "status", Aria: map[string]string{"live": "polite"}, Style: calloutStageStyle()},
			html.Div(html.Props{Style: calloutPanelStyle()},
				html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "25px", "letter-spacing": ".2em", "text-transform": "uppercase"}}, ui.Text(T("en", "dm.callout.steering", nil))),
				html.P(html.Props{Style: map[string]string{"margin": "12px 0 0", "color": "#efe6d2", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "32px", "line-height": "1.2", "text-shadow": "0 2px 8px #000"}}, ui.Text(view.Text)),
			),
		)
	}
}

func calloutStageStyle() map[string]string {
	return map[string]string{"position": "absolute", "inset": "0", "z-index": "2", "pointer-events": "none", "color": "#efe6d2"}
}

// calloutPanelStyle keeps steering above the scene's characters and caption.
func calloutPanelStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "left": "50%", "top": "22px", "transform": "translateX(-50%)",
		"width": "860px", "padding": "20px 30px", "box-sizing": "border-box",
		"border": "1px solid rgba(184,137,58,.55)", "border-radius": "16px",
		"background": "linear-gradient(180deg,rgba(10,15,22,.96),rgba(8,12,18,.92))", "backdrop-filter": "blur(3px)",
		"box-shadow": "0 12px 30px rgba(0,0,0,.4)",
		"text-align": "center",
	}
}
