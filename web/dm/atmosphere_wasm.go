//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
)

// AtmosphereComponent is the shared mood overlay for every painted TV scene
// screen (opening, exploration, conversation, check, resolution, hook and
// cliffhanger): a soft vignette, a faint film-grain texture, and a slow
// drifting river-fog band along the bottom. It takes no props and never
// changes between renders, so it costs one static DOM subtree per layer; its
// only per-frame work is a CSS transform animation on the fog band (skipped
// entirely under prefers-reduced-motion, see dmAtmosphereCSS).
func AtmosphereComponent() router.Component {
	return func(_ router.Attrs) *router.Element {
		return html.Div(html.Props{Class: "df-dm-atmosphere", Aria: map[string]string{"hidden": "true"}},
			html.Div(html.Props{Class: "df-dm-atmosphere-vignette"}),
			html.Div(html.Props{Class: "df-dm-atmosphere-grain"}),
			html.Div(html.Props{Class: "df-dm-atmosphere-fog"}),
		)
	}
}
