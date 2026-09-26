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
			joined = append(joined, waitingSeat(seat, locale))
		}
		if len(joined) == 0 {
			joined = append(joined, html.Li(html.Props{Style: map[string]string{"padding": "13px", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "16px", "text-align": "center"}}, html.Text("Your companions will appear here.")))
		}
		return html.Section(html.Props{Class: "df-phone-waiting", Role: "main", Style: waitingContentStyle()},
			html.Div(html.Props{Style: map[string]string{"padding": "22px 10px 12px", "text-align": "center"}},
				html.Div(html.Props{Style: map[string]string{"width": "58px", "height": "58px", "margin": "0 auto 13px", "display": "grid", "place-items": "center", "border": "1px solid #d9a441", "border-radius": "50%", "box-shadow": "inset 0 0 18px rgba(217,164,65,.18), 0 0 18px rgba(217,164,65,.12)", "color": "#e7c27a", "font-family": "Georgia, serif", "font-size": "28px"}}, html.Text("✦")),
				html.P(html.Props{Style: map[string]string{"margin": "0 0 4px", "color": "#d9a441", "font-size": "10px", "font-weight": "700", "letter-spacing": ".2em"}}, html.Text("THE DROWNED LANTERN")),
				html.H1(html.Props{Class: "df-phone-waiting-title", Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "29px", "line-height": "1.05"}}, html.Text("Welcome, "+model.PlayerName)),
				html.P(html.Props{Class: "df-phone-waiting-seat-number", Style: map[string]string{"margin": "7px 0 0", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px"}}, html.Text(WaitingSeatLabel(locale, model.PlayerNumber))),
			),
			html.Section(html.Props{Class: "df-phone-waiting-panel", Style: map[string]string{"padding": "13px", "border": "1px solid rgba(217,164,65,.45)", "border-radius": "12px", "background": "rgba(17,21,29,.84)", "box-shadow": "inset 0 0 20px rgba(217,164,65,.06)"}},
				html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "align-items": "center", "padding-bottom": "9px", "border-bottom": "1px solid rgba(168,159,140,.2)"}}, html.Span(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "18px"}}, html.Text("At the table")), html.Small(html.Props{Role: "status", Style: map[string]string{"color": "#a89f8c", "font-size": "11px"}}, html.Text(joinedSummary(locale, len(model.Joined))))),
				html.Ul(html.Props{Class: "df-phone-waiting-list", Aria: map[string]string{"label": "Joined players"}, Style: map[string]string{"display": "grid", "gap": "7px", "margin": "10px 0 0", "padding": "0", "list-style": "none"}}, joined...),
			),
			html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "padding": "14px 8px 8px", "text-align": "center"}}, html.Div(html.Props{Style: map[string]string{"width": "50px", "height": "2px", "margin": "0 auto 11px", "background": "#d9a441", "box-shadow": "0 0 12px rgba(217,164,65,.45)"}}), html.P(html.Props{Class: "df-phone-waiting-status", Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px"}}, html.Text(WaitingStatus(locale)))),
		)
	}
}

func waitingSeat(seat WaitingSeat, locale string) ui.Node {
	return html.Li(html.Props{Class: "df-phone-waiting-seat", Style: map[string]string{"min-height": "48px", "box-sizing": "border-box", "display": "flex", "align-items": "center", "justify-content": "space-between", "padding": "8px 11px", "border": "1px solid rgba(168,159,140,.28)", "border-radius": "8px", "background": "rgba(27,31,41,.82)", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px"}}, html.Span(html.Props{}, html.Text(seat.Name)), html.Small(html.Props{Style: map[string]string{"color": "#a89f8c", "font-family": "Inter, system-ui, sans-serif", "font-size": "10px", "letter-spacing": ".06em", "text-transform": "uppercase"}}, html.Text(WaitingSeatLabel(locale, seat.Number))))
}

func waitingContentStyle() map[string]string {
	return map[string]string{"min-height": "100%", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "gap": "12px", "padding": "4px 0 2px", "background": "radial-gradient(circle at 50% 0%, #2a2933 0, #151a24 42%, #0b0f16 100%)", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif"}
}
