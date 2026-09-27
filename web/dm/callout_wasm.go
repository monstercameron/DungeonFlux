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
			html.Div(html.Props{Style: calloutBackdropStyle()}),
			html.Div(html.Props{Style: calloutPanelStyle()},
				html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel,'Cormorant Garamond',Georgia,serif", "font-size": "25px", "letter-spacing": ".2em", "text-transform": "uppercase"}}, ui.Text(T("en", "dm.callout.steering", nil))),
				html.Div(html.Props{Style: map[string]string{"width": "600px", "max-width": "80%", "height": "1px", "margin": "18px auto 22px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
				html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "54px", "line-height": "1.15", "text-shadow": "0 2px 8px #000"}}, ui.Text(view.Text)),
				html.Div(html.Props{Style: map[string]string{"margin-top": "26px", "color": "#a89f8c", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "23px", "font-style": "italic"}}, ui.Text(T("en", "dm.callout.prompt", nil))),
			),
		)
	}
}

func calloutStageStyle() map[string]string {
	return map[string]string{"position": "absolute", "inset": "0", "z-index": "2", "pointer-events": "none", "color": "#efe6d2"}
}

func calloutBackdropStyle() map[string]string {
	style := map[string]string{"position": "absolute", "inset": "0", "background": "linear-gradient(180deg,rgba(8,10,15,.2),rgba(8,10,15,.82)),rgba(8,10,15,.6)"}
	if artURL := ArtURL("ui/banner_callout"); artURL != "" {
		style["background-image"] = "linear-gradient(180deg,rgba(8,10,15,.2),rgba(8,10,15,.82)),url('" + artURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return style
}

func calloutPanelStyle() map[string]string {
	return map[string]string{"position": "absolute", "left": "270px", "right": "270px", "top": "355px", "min-height": "315px", "padding": "58px 90px 50px", "box-sizing": "border-box", "border": "1px solid #b8893a", "border-radius": "12px", "background": "rgba(12,18,28,.88)", "box-shadow": "0 20px 52px rgba(0,0,0,.6), inset 0 0 0 1px rgba(12,12,16,.78)", "text-align": "center"}
}
