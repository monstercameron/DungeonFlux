//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const titleSubtitle = "AI DUNGEON MASTER FOR FIFTH-EDITION FANTASY"

// LobbyComponent renders the fixed-canvas title and room lobby.
func LobbyComponent(model LobbyModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		art := titleArtFor(currentAspectClass(), ArtURL)
		return html.Main(html.Props{Class: "df-dm-lobby", Role: "main", Style: lobbySurfaceStyle()},
			titlePlateAt(art.Wordmark), lobbyTagline(), lobbyStatus(model), lobbyQuote(),
			lobbyJoinPanel(model, art), lobbyPartyPanel(model, art), lobbyTablePanel(art), lobbyFooter(),
		)
	}
}

func lobbySurfaceStyle() map[string]string {
	return map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif"}
}

func titlePlateAt(wordmark string) ui.Node {
	return html.Div(html.Props{Class: "df-lobby-title-plate", Style: absoluteStyle(250, 60, 750, 205)}, TitlePlate(wordmark, "DungeonFlux", titleSubtitle))
}

func lobbyTagline() ui.Node {
	return html.Div(html.Props{Class: "df-lobby-tagline", Style: absoluteStyle(50, 55, 210, 95)}, html.P(html.Props{}, ui.Text("STORIES RESPOND.\nWORLDS EVOLVE.\nYOU BELONG.")))
}

func lobbyStatus(model LobbyModel) ui.Node {
	return html.Div(html.Props{Class: "df-lobby-status-stack", Style: map[string]string{"position": "absolute", "left": "390px", "top": "384px", "width": "500px", "z-index": "4", "display": "grid", "gap": "10px"}},
		GoldPlateButton("✦", LobbyStatus(model)),
		DarkButton("↗", "Join: "+shortJoinURL(model.JoinURL)),
		DarkButton("⌂", "Host controls on the host page"),
		DarkButton("文", "Locale: "+strings.ToUpper(localeOrDefault(model.Locale))),
	)
}

func shortJoinURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/p"
	}
	return value
}

func lobbyQuote() ui.Node {
	return html.Div(html.Props{Class: "df-lobby-quote", Style: absoluteStyle(890, 520, 350, 78)}, html.P(html.Props{}, ui.Text("“Same table.\nHigher possibilities.”")))
}

func lobbyJoinPanel(model LobbyModel, art titleArt) ui.Node {
	qr := lobbyQR(model.QRURL)
	content := []ui.Node{
		html.Div(html.Props{Class: "df-lobby-qr-wrap", Style: map[string]string{"position": "absolute", "left": "30px", "top": "55px", "width": "185px", "height": "185px"}}, qr),
		html.Div(html.Props{Class: "df-lobby-code-box", Style: map[string]string{"position": "absolute", "left": "235px", "top": "98px", "width": "250px", "height": "76px"}}, html.Span(html.Props{Class: "df-lobby-code-label"}, ui.Text("ROOM CODE")), html.Strong(html.Props{Class: "df-lobby-code-value"}, ui.Text(SpacedRoomCode(model.RoomCode)))),
		html.P(html.Props{Class: "df-lobby-scan-copy", Style: map[string]string{"position": "absolute", "left": "30px", "right": "30px", "bottom": "16px"}}, ui.Text("▣  Scan with your phone to join the game.")),
	}
	return panelAt("PLAYERS JOIN HERE", 40, 665, 520, 320, art.PanelFrame, content...)
}

func lobbyQR(value string) ui.Node {
	value = strings.TrimSpace(value)
	if resolved := ArtURL(value); resolved != "" {
		value = resolved
	}
	if value == "" {
		return html.Div(html.Props{Class: "df-lobby-qr-empty", Role: "img", Aria: map[string]string{"label": "Join QR code loading"}}, ui.Text("QR"))
	}
	return html.Img(html.Props{Class: "df-lobby-qr-image", Src: value, Alt: "Scan to join the game"})
}

func lobbyPartyPanel(model LobbyModel, art titleArt) ui.Node {
	card := func(seat Seat) ui.Node {
		return PortraitCard(PortraitCardModel{Name: seat.Name, Subtitle: SeatSubtitle(seat), Flavor: seat.Flavor, PortraitURL: seat.PortraitURL, Joined: seat.Joined, Ready: seat.Ready})
	}
	content := []ui.Node{
		html.Div(html.Props{Class: "df-lobby-party-cards", Role: "list", Style: map[string]string{"position": "absolute", "left": "25px", "right": "25px", "top": "50px", "height": "230px", "display": "grid", "grid-template-columns": "1fr 1fr", "gap": "15px"}}, card(model.Seats[0]), card(model.Seats[1])),
		html.P(html.Props{Class: "df-lobby-party-footer", Style: map[string]string{"position": "absolute", "left": "25px", "right": "25px", "bottom": "14px"}}, ui.Text("▣  Players use their phones as character sheets, dice rollers, and controllers.")),
	}
	return panelAt("YOUR PARTY", 580, 665, 740, 320, art.PanelFrame, content...)
}

func lobbyTablePanel(art titleArt) ui.Node {
	features := [][2]string{{"◈", "AI Narration"}, {"∿", "Voice NPCs"}, {"◇", "Rules Engine"}, {"✺", "Character Memory"}, {"♫", "Dynamic Music"}, {"▣", "Cliffhanger Clips"}}
	items := make([]ui.Node, 0, len(features))
	for _, feature := range features {
		items = append(items, featureTile(feature[0], feature[1]))
	}
	content := []ui.Node{html.Div(html.Props{Class: "df-table-grid", Style: map[string]string{"position": "absolute", "left": "25px", "right": "25px", "top": "52px", "bottom": "25px", "display": "grid", "grid-template-columns": "repeat(3, 1fr)", "grid-template-rows": "repeat(2, 1fr)", "gap": "12px"}}, items...)}
	return panelAt("THE TABLE", 1340, 665, 540, 320, art.PanelFrame, content...)
}

func panelAt(title string, left, top, width, height int, frame string, children ...ui.Node) ui.Node {
	panel := OrnatePanel(title, children...)
	if frame == "" {
		return html.Div(html.Props{Class: "df-lobby-panel", Style: absoluteStyle(left, top, width, height)}, panel)
	}
	// The ornate frame art is a 9-slice border image on its own layer behind
	// the glass panel (GWC style maps drop custom properties, so no CSS var).
	layer := html.Div(html.Props{Class: "df-panel-frame", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"border-image-source": "url('" + frame + "')"}})
	return html.Div(html.Props{Class: "df-lobby-panel is-framed", Style: absoluteStyle(left, top, width, height)}, layer, panel)
}

func absoluteStyle(left, top, width, height int) map[string]string {
	return map[string]string{"position": "absolute", "left": strconv.Itoa(left) + "px", "top": strconv.Itoa(top) + "px", "width": strconv.Itoa(width) + "px", "height": strconv.Itoa(height) + "px"}
}

func lobbyFooter() ui.Node {
	return html.Div(html.Props{Class: "df-lobby-footer", Style: map[string]string{"position": "absolute", "left": "40px", "right": "40px", "top": "1005px", "display": "flex", "justify-content": "space-between"}}, html.Span(html.Props{}, ui.Text("NO APP. NO ACCOUNT. SCAN AND PLAY.")), html.Span(html.Props{}, ui.Text("SAME GAME. A BRIGHTER TOMORROW.")))
}
