//go:build js && wasm

package dm

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// LobbyComponent is the GWC route component for the DM lobby.
func LobbyComponent(model LobbyModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(model.Locale)
		return html.Main(html.Props{Class: "df-dm-lobby", Role: "main"},
			html.P(html.Props{Class: "df-eyebrow"}, ui.Text(T(locale, "dm.eyebrow_live", nil))),
			html.H1(html.Props{}, ui.Text(LobbyTitle(locale))),
			html.P(html.Props{Class: "df-lobby-intro"}, ui.Text(LobbyIntro(locale))),
			html.Section(html.Props{Class: "df-lobby-join"},
				html.Img(html.Props{Src: model.QRURL, Alt: T(locale, "dm.qr_alt", nil), Class: "df-lobby-qr"}),
				html.Div(html.Props{Class: "df-lobby-code"},
					html.Small(html.Props{}, ui.Text(RoomCodeLabel(locale))),
					html.Code(html.Props{}, ui.Text(model.RoomCode)),
				),
			),
			html.Section(html.Props{Class: "df-lobby-seats", Role: "list", Aria: map[string]string{"label": SeatsLabel(locale)}}, seatCard(locale, model.Seats[0]), seatCard(locale, model.Seats[1])),
			html.Section(html.Props{Class: "df-lobby-audio", Aria: map[string]string{"live": "polite"}},
				html.H2(html.Props{}, ui.Text(ListenLabel(locale))),
				html.P(html.Props{}, ui.Text(model.AudioState)),
				html.Audio(html.Props{ID: "df-opening-audio", Hidden: true, Src: model.OpeningAudio, Raw: map[string]any{"controls": true}}),
			),
		)
	}
}

func seatCard(locale string, seat Seat) ui.Node {
	return html.Div(html.Props{Class: "df-lobby-seat", Role: "listitem"},
		html.H2(html.Props{}, ui.Text(SeatName(locale, seat.Name, seat.Number))),
		html.P(html.Props{}, ui.Text(SeatStatus(locale, seat.Joined, seat.Ready))),
	)
}
