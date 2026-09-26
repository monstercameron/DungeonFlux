//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// EndCardComponent renders the terminal end card for the DM screen.
func EndCardComponent(model EndCardModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Main(html.Props{Class: "df-dm-end-card", Role: "main"},
			html.Div(html.Props{Class: "df-dm-end-card-content"},
				html.P(html.Props{Class: "df-eyebrow"}, ui.Text(T(locale, "dm.eyebrow_end", nil))),
				html.H1(html.Props{}, ui.Text(model.Title)),
				html.P(html.Props{Class: "df-dm-end-card-subtitle"}, ui.Text(model.Subtitle)),
				html.Section(html.Props{Class: "df-dm-end-card-attribution", Aria: map[string]string{"label": "SRD attribution"}},
					html.H2(html.Props{}, ui.Text(EndRules(locale))),
					html.P(html.Props{}, ui.Text(model.Attribution)),
				),
			),
		)
	}
}
