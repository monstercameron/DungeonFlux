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
		return html.Div(html.Props{Class: "df-dm-callout", Hidden: !view.Visible, Role: "status", Aria: map[string]string{"live": "polite"}},
			html.P(html.Props{}, ui.Text(view.Text)),
		)
	}
}
