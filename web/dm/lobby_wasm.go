//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// LobbyComponent is the GWC route component for the DM lobby.
func LobbyComponent(model LobbyModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Main(html.Props{Class: "df-dm-lobby", Role: "main", Style: lobbyStyle()},
			html.Div(html.Props{Class: "df-lobby-header", Style: headerStyle()},
				html.P(html.Props{Class: "df-eyebrow", Style: eyebrowStyle()}, ui.Text(T(locale, "dm.eyebrow_live", nil))),
				html.H1(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(3rem, 6vw, 6rem)", "line-height": "1"}}, ui.Text(LobbyTitle(locale))),
				html.P(html.Props{Class: "df-lobby-intro", Style: map[string]string{"max-width": "48rem", "margin": "1rem 0 0", "color": "#a89f8c", "font-size": "clamp(1.25rem, 2vw, 2rem)"}}, ui.Text(LobbyIntro(locale))),
			),
			html.Div(html.Props{Class: "df-lobby-grid", Style: gridStyle()},
				joinPanel(locale, model),
				html.Div(html.Props{Class: "df-lobby-roster", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "1rem"}},
					html.H2(html.Props{Style: sectionHeadingStyle()}, ui.Text(SeatsLabel(locale))),
					html.Section(html.Props{Class: "df-lobby-seats", Role: "list", Aria: map[string]string{"label": SeatsLabel(locale)}, Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(2, minmax(0, 1fr))", "gap": "1rem"}}, seatCard(locale, model.Seats[0]), seatCard(locale, model.Seats[1])),
					html.Section(html.Props{Class: "df-lobby-audio", Aria: map[string]string{"live": "polite"}, Style: audioStyle()},
						html.H2(html.Props{Style: map[string]string{"margin": "0 0 .4rem", "font-size": "1rem", "color": "#d9a441", "text-transform": "uppercase", "letter-spacing": ".12em"}}, ui.Text(ListenLabel(locale))),
						html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#a89f8c"}}, ui.Text(model.AudioState)),
						html.Audio(html.Props{ID: "df-opening-audio", Hidden: true, Src: model.OpeningAudio, Raw: map[string]any{"controls": true}}),
					),
				),
			),
		)
	}
}

func lobbyStyle() map[string]string {
	return map[string]string{"min-height": "100vh", "box-sizing": "border-box", "padding": "clamp(2rem, 5vw, 6rem)", "background": "radial-gradient(circle at 76% 18%, #292a32 0, #171922 42%, #0f1117 100%)", "color": "#efe6d2", "font-family": "Arial, sans-serif"}
}

func headerStyle() map[string]string {
	return map[string]string{"max-width": "78rem", "margin": "0 auto clamp(2rem, 5vw, 4rem)"}
}

func eyebrowStyle() map[string]string {
	return map[string]string{"margin": "0 0 1rem", "color": "#d9a441", "font-size": "1rem", "font-weight": "700", "letter-spacing": ".18em"}
}

func gridStyle() map[string]string {
	return map[string]string{"max-width": "78rem", "margin": "0 auto", "display": "grid", "grid-template-columns": "minmax(0, .85fr) minmax(0, 1.15fr)", "gap": "clamp(2rem, 5vw, 6rem)", "align-items": "start"}
}

func sectionHeadingStyle() map[string]string {
	return map[string]string{"margin": "0 0 .8rem", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "2rem"}
}

func audioStyle() map[string]string {
	return map[string]string{"margin-top": "1rem", "padding": "1rem 1.25rem", "border-left": "3px solid #3aa39a", "background": "rgba(58,163,154,.08)"}
}

func joinPanel(locale string, model LobbyModel) ui.Node {
	return html.Section(html.Props{Class: "df-lobby-join", Aria: map[string]string{"label": RoomCodeLabel(locale)}, Style: map[string]string{"display": "grid", "grid-template-columns": "minmax(12rem, 18rem) 1fr", "gap": "clamp(1.25rem, 3vw, 2.5rem)", "align-items": "center", "padding": "clamp(1rem, 3vw, 2rem)", "border": "1px solid #8f702f", "border-radius": "14px", "background": "rgba(26,29,38,.88)", "box-shadow": "inset 0 0 30px rgba(0,0,0,.25)"}},
		html.Img(html.Props{Src: model.QRURL, Alt: T(locale, "dm.qr_alt", nil), Class: "df-lobby-qr", Style: map[string]string{"display": "block", "width": "100%", "max-width": "18rem", "aspect-ratio": "1", "padding": ".7rem", "box-sizing": "border-box", "background": "#efe6d2", "border-radius": "8px"}}),
		html.Div(html.Props{Class: "df-lobby-code", Style: map[string]string{"min-width": "0"}},
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-bottom": ".6rem", "color": "#d9a441", "font-size": "1rem", "font-weight": "700", "letter-spacing": ".15em"}}, ui.Text(RoomCodeLabel(locale))),
			html.Code(html.Props{Style: map[string]string{"display": "block", "color": "#efe6d2", "font-size": "clamp(3.5rem, 8vw, 7rem)", "font-weight": "700", "letter-spacing": ".16em", "line-height": "1", "overflow-wrap": "anywhere"}}, ui.Text(model.RoomCode)),
			html.A(html.Props{Href: model.JoinURL, Class: "df-lobby-url", Style: map[string]string{"display": "block", "margin-top": "1.4rem", "color": "#a89f8c", "font-size": "clamp(1rem, 1.7vw, 1.4rem)", "overflow-wrap": "anywhere"}, Aria: map[string]string{"label": "Player join URL"}}, ui.Text(model.JoinURL)),
		),
	)
}

func seatCard(locale string, seat Seat) ui.Node {
	status := SeatStatus(locale, seat.Joined, seat.Ready)
	border := "#4b4c54"
	if seat.Joined {
		border = "#3aa39a"
	}
	return html.Div(html.Props{Class: "df-lobby-seat", Role: "listitem", Style: map[string]string{"min-height": "9rem", "padding": "1.25rem", "box-sizing": "border-box", "border": "1px solid " + border, "border-radius": "12px", "background": "rgba(15,17,23,.72)"}, Raw: map[string]any{"data-seat": seat.Number, "data-joined": seat.Joined, "data-ready": seat.Ready}},
		html.P(html.Props{Style: map[string]string{"margin": "0 0 1rem", "color": "#a89f8c", "font-size": "1rem", "font-weight": "700", "letter-spacing": ".12em"}}, ui.Text("SEAT "+strconv.Itoa(seat.Number))),
		html.H2(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1.5rem, 3vw, 2.5rem)"}}, ui.Text(SeatName(locale, seat.Name, seat.Number))),
		html.P(html.Props{Style: map[string]string{"margin": ".7rem 0 0", "color": "#3aa39a", "font-size": "1.15rem", "font-weight": "700"}, Aria: map[string]string{"label": "Seat status"}}, ui.Text(status)),
	)
}
