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

// calloutBackdropStyle dims just enough for the text card's contrast; the
// banner art (or, absent it, the scene beneath) stays the visible backdrop
// instead of a flat opaque colour.
func calloutBackdropStyle() map[string]string {
	style := map[string]string{"position": "absolute", "inset": "0", "background-color": "rgba(8,10,15,.28)", "background-image": "linear-gradient(180deg,rgba(8,10,15,.06),rgba(8,10,15,.38))"}
	if artURL := ArtURL("ui/banner_callout"); artURL != "" {
		style["background-image"] = "linear-gradient(180deg,rgba(8,10,15,.06),rgba(8,10,15,.4)),url('" + artURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return style
}

// calloutPanelStyle is a narrower, feathered card (a radial mask fades its
// left and right edges to transparent) so the scroll-banner backdrop reads
// around it instead of being covered by a flat rectangle with hard side
// edges.
func calloutPanelStyle() map[string]string {
	mask := "radial-gradient(ellipse 96% 100% at 50% 50%, #000 80%, transparent 100%)"
	return map[string]string{
		"position": "absolute", "left": "50%", "top": "370px", "transform": "translateX(-50%)",
		"width": "min(1120px, 78%)", "min-height": "300px", "padding": "56px 88px 46px", "box-sizing": "border-box",
		"border": "1px solid rgba(184,137,58,.55)", "border-radius": "16px",
		"background": "linear-gradient(180deg,rgba(10,15,22,.6),rgba(8,12,18,.72))", "backdrop-filter": "blur(3px) saturate(1.08)",
		"box-shadow": "0 24px 60px rgba(0,0,0,.5)", "-webkit-mask-image": mask, "mask-image": mask,
		"text-align": "center",
	}
}
