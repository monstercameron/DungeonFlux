//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// creationDualSeatPanels gives both players equal space on the shared TV.
// Phone controls remain on the phones; the TV is a calm, readable comparison
// of each seat's current choices and rolled result.
func creationDualSeatPanels(seats [2]CreationSeat) ui.Node {
	items := make([]ui.Node, 0, len(seats))
	for index, seat := range seats {
		items = append(items, creationDualSeatPanel(seat, index == 0))
	}
	return html.Div(html.Props{Class: "df-dm-creation-dual", Style: map[string]string{
		"position": "absolute", "left": "130px", "right": "130px", "top": "214px", "height": "748px",
		"z-index": "2", "display": "grid", "grid-template-columns": "1fr 1fr", "gap": "30px",
	}}, items...)
}

func creationDualSeatPanel(seat CreationSeat, first bool) ui.Node {
	accent := "#d9a441"
	if !first {
		accent = "#62b9aa"
	}
	style := creationPanelStyle("0", "0", "100%", "100%")
	style["position"], style["left"], style["top"] = "relative", "auto", "auto"
	style["height"], style["padding"], style["box-sizing"] = "100%", "22px", "border-box"
	style["border-color"] = accent
	style["box-shadow"] = "0 18px 45px rgba(0,0,0,.56), inset 0 0 34px rgba(70,130,120,.07)"
	identity := creationDualIdentity(seat, accent)
	portrait := creationHeroStandIn(seat)
	portraitNode := html.Div(html.Props{Style: map[string]string{
		"width": "252px", "height": "430px", "flex": "0 0 252px", "overflow": "hidden",
		"border": "2px solid " + accent, "border-radius": "8px", "background": "radial-gradient(circle at 50% 25%,#304963,#0a0e15)",
	}}, html.Img(html.Props{Src: portrait, Alt: creationDisplayValue(seat.Name, "Hero portrait"), Hidden: portrait == "", Style: map[string]string{
		"width": "100%", "height": "100%", "object-fit": "cover", "object-position": "center 12%",
	}}))
	details := html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1", "display": "flex", "flex-direction": "column", "gap": "14px"}},
		creationDualChoiceLine("SPECIES", seat.Species, "Awaiting choice"),
		creationDualChoiceLine("GENDER", seat.Gender, "Awaiting choice"),
		creationDualChoiceLine("CLASS", seat.Class, "Engine will roll"),
		creationDualStats(seat),
		creationDualStatus(seat, accent),
	)
	return html.Article(html.Props{Class: "df-dm-creation-seat", Role: "group", Aria: map[string]string{"label": "Player " + strconv.Itoa(int(seat.Number)) + " character"}, Style: style}, identity,
		html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "24px", "height": "650px", "align-items": "stretch"}}, portraitNode, details),
	)
}

func creationDualIdentity(seat CreationSeat, accent string) ui.Node {
	name := creationDisplayValue(seat.Name, "Player "+strconv.Itoa(int(seat.Number)))
	status := creationDisplayValue(seat.Status, "Waiting for player")
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "baseline", "justify-content": "space-between", "gap": "16px", "margin-bottom": "14px", "border-bottom": "1px solid rgba(217,164,65,.38)", "padding-bottom": "12px"}},
		html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.Small(html.Props{Style: map[string]string{"display": "block", "color": accent, "font-size": "14px", "letter-spacing": ".18em", "font-weight": "700"}}, html.Text("PLAYER "+strconv.Itoa(int(seat.Number)))), html.H2(html.Props{Style: map[string]string{"margin": "4px 0 0", "font-family": "Cinzel,Georgia,serif", "font-size": "30px", "font-weight": "500", "overflow": "hidden", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(name))),
		html.Span(html.Props{Role: "status", Style: map[string]string{"flex": "0 0 auto", "padding": "7px 12px", "border": "1px solid " + accent, "border-radius": "999px", "color": accent, "font-size": "14px", "font-weight": "700", "letter-spacing": ".06em", "white-space": "nowrap"}}, html.Text(strings.ToUpper(status))),
	)
}

func creationDualChoiceLine(label, value, fallback string) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "12px", "border-bottom": "1px solid rgba(200,189,168,.16)", "padding": "5px 0"}},
		html.Small(html.Props{Style: map[string]string{"color": "#a89f8c", "font-size": "13px", "letter-spacing": ".12em"}}, html.Text(label)),
		html.Strong(html.Props{Style: map[string]string{"max-width": "65%", "overflow": "hidden", "text-overflow": "ellipsis", "white-space": "nowrap", "font-family": "Cormorant Garamond,Georgia,serif", "font-size": "20px", "color": "#efe6d2"}}, html.Text(creationDisplayValue(value, fallback))),
	)
}

func creationDualStats(seat CreationSeat) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"margin-top": "4px", "padding": "12px", "border": "1px solid rgba(184,137,58,.55)", "border-radius": "7px", "background": "rgba(7,13,21,.58)"}},
		html.Small(html.Props{Style: map[string]string{"display": "block", "margin-bottom": "8px", "color": "#e7c27a", "font-size": "13px", "letter-spacing": ".14em", "font-weight": "700"}}, html.Text("ROLLED HERO")),
		creationStatGrid(seat), creationSecondaryRow(seat),
	)
}

func creationDualStatus(seat CreationSeat, accent string) ui.Node {
	label := "Waiting for phone"
	if seat.HasStats {
		label = "Confirm on your phone"
	}
	if seat.Ready {
		label = "Ready for adventure"
	}
	return html.Div(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"margin-top": "auto", "padding": "13px", "border": "1px solid " + accent, "border-radius": "7px", "background": "rgba(15,30,35,.66)", "text-align": "center", "color": accent, "font-family": "Cinzel,Georgia,serif", "font-size": "20px"}}, html.Text("✦  "+label))
}
