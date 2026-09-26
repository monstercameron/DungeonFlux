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
		return html.Main(html.Props{Class: "df-dm-lobby", Role: "main"},
			html.P(html.Props{Class: "df-eyebrow"}, ui.Text("DUNGEONFLUX · LIVE TABLE")),
			html.H1(html.Props{}, ui.Text("Gather at the table")),
			html.P(html.Props{Class: "df-lobby-intro"}, ui.Text("Players joined by scanning this code. No accounts, no app.")),
			html.Section(html.Props{Class: "df-lobby-join"},
				html.Img(html.Props{Src: model.QRURL, Alt: "Scan to join the DungeonFlux table", Class: "df-lobby-qr"}),
				html.Div(html.Props{Class: "df-lobby-code"},
					html.Small(html.Props{}, ui.Text("ROOM CODE")),
					html.Code(html.Props{}, ui.Text(model.RoomCode)),
				),
			),
			html.Section(html.Props{Class: "df-lobby-seats", Role: "list", Aria: map[string]string{"label": "Player seats"}}, seatCard(model.Seats[0]), seatCard(model.Seats[1])),
			html.Section(html.Props{Class: "df-lobby-audio", Aria: map[string]string{"live": "polite"}},
				html.H2(html.Props{}, ui.Text("Listen")),
				html.P(html.Props{}, ui.Text(model.AudioState)),
				html.Audio(html.Props{ID: "df-opening-audio", Hidden: true, Src: model.OpeningAudio, Raw: map[string]any{"controls": true}}),
			),
		)
	}
}

func seatCard(seat Seat) ui.Node {
	status := "Waiting to join"
	if seat.Joined {
		status = "Joined"
	}
	if seat.Ready {
		status = "Ready"
	}
	return html.Div(html.Props{Class: "df-lobby-seat", Role: "listitem"},
		html.H2(html.Props{}, ui.Text(SeatLabel(seat))),
		html.P(html.Props{}, ui.Text(status)),
	)
}
