//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// PhoneFrame renders the shared header, status, content region, and bottom
// action affordance around a phone screen. Callers can use it when composing
// a screen in the shell without duplicating accessibility or theme tokens.
func PhoneFrame(model FrameModel, content ui.Node, action ui.Node) router.Component {
	return func(_ router.Attrs) *router.Element {
		theme := DefaultPhoneTheme()
		status := ConnectionLabel(model.Locale, model.Connection)
		statusClass := "df-phone-connection df-phone-connection-" + string(model.Connection)
		if model.Connection == "" {
			statusClass = "df-phone-connection df-phone-connection-offline"
		}
		header := html.Header(html.Props{Class: "df-phone-header"},
			html.Div(html.Props{Class: "df-phone-brand"},
				html.Span(html.Props{Class: "df-phone-eyebrow"}, html.Text("DUNGEONFLUX")),
				html.Span(html.Props{Class: "df-phone-seat"}, html.Text(model.DisplayName))),
			html.Span(html.Props{Class: statusClass, Role: "status", Aria: map[string]string{"live": "polite"}}, html.Text(status)),
		)
		body := html.Section(html.Props{Class: "df-phone-frame-content", Role: "region", Aria: map[string]string{"label": model.Title}}, content)
		footer := html.Footer(html.Props{Class: "df-phone-action", Style: map[string]string{"min-height": theme.TouchTarget}}, action)
		return html.Main(html.Props{Class: "df-phone df-phone-frame", Style: phoneFrameStyle(theme)}, header, body, footer)
	}
}

func phoneFrameStyle(theme PhoneTheme) map[string]string {
	return map[string]string{"background": theme.Ink, "color": theme.Parchment, "font-family": "Inter, ui-sans-serif, system-ui, sans-serif", "min-height": "100vh", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "padding": "16px", "gap": "12px", "transition": "opacity " + theme.Transition}
}
