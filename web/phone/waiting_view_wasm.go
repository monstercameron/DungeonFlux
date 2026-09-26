//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// WaitingScreen renders the welcoming portrait lobby state.
func WaitingScreen(model WaitingModel, locale string) router.Component {
	return func(_ router.Attrs) *router.Element {
		joined := make([]ui.Node, 0, len(model.Joined))
		for _, seat := range model.Joined {
			joined = append(joined, html.Li(html.Props{Class: "df-phone-waiting-seat"}, html.Span(html.Props{}, html.Text(seat.Name)), html.Small(html.Props{}, html.Text(WaitingSeatLabel(locale, seat.Number)))))
		}
		name := html.H1(html.Props{Class: "df-phone-waiting-title"}, html.Text("Welcome, "+model.PlayerName))
		seat := html.P(html.Props{Class: "df-phone-waiting-seat-number"}, html.Text(WaitingSeatLabel(locale, model.PlayerNumber)))
		people := html.P(html.Props{Class: "df-phone-waiting-joined", Role: "status"}, html.Text(joinedSummary(locale, len(model.Joined))))
		return html.Main(html.Props{Class: "df-phone df-phone-waiting", Role: "main", Style: waitingStyle()}, name, seat, people, html.Ul(html.Props{Class: "df-phone-waiting-list", Aria: map[string]string{"label": "Joined players"}}, joined...), html.P(html.Props{Class: "df-phone-waiting-status", Role: "status", Aria: map[string]string{"live": "polite"}}, html.Text(WaitingStatus(locale))))
	}
}

func waitingStyle() map[string]string {
	return map[string]string{"box-sizing": "border-box", "min-height": "100vh", "width": "100%", "max-width": "480px", "margin": "0 auto", "padding": "48px 22px calc(32px + env(safe-area-inset-bottom))", "display": "flex", "flex-direction": "column", "align-items": "center", "gap": "14px", "background": "radial-gradient(circle at 50% -10%, #342b20 0, #171921 46%, #0f1117 100%)", "color": "#efe6d2", "font-family": "system-ui, sans-serif", "text-align": "center"}
}
