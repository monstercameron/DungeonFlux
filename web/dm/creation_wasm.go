//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CreationComponent renders both players' live character-building progress.
func CreationComponent(model CreationModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{Class: "df-dm-creation", Role: "region", Aria: map[string]string{"label": "Character creation"}, Style: creationStyle()},
			html.P(html.Props{Class: "df-dm-creation-eyebrow", Style: map[string]string{"margin": "0 0 .5rem", "color": "#d9a441", "font-size": "clamp(.8rem,1.2vw,1.25rem)", "font-weight": "700", "letter-spacing": ".18em", "text-transform": "uppercase"}}, ui.Text("CHARACTER CREATION")),
			html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia,serif", "font-size": "clamp(2.6rem,5vw,6rem)", "line-height": "1.05"}}, ui.Text("Build your heroes")),
			html.P(html.Props{Style: map[string]string{"margin": ".7rem 0 0", "color": "#a89f8c", "font-size": "clamp(1.15rem,2vw,2rem)"}}, ui.Text(model.Prompt)),
			html.Div(html.Props{Class: "df-dm-creation-seats", Role: "list", Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(2,minmax(0,1fr))", "gap": "clamp(1rem,2.5vw,3rem)", "margin-top": "clamp(1.5rem,4vw,4rem)"}}, seatProgress(model.Seats[0]), seatProgress(model.Seats[1])),
		)
	}
}

func creationStyle() map[string]string {
	return map[string]string{"height": "100%", "min-height": "100%", "padding": "clamp(2rem,6vw,7rem)", "background": "radial-gradient(circle at 50% 15%,rgba(57,49,35,.48),transparent 58%),linear-gradient(135deg,rgba(15,17,23,.98),rgba(26,29,38,.94))", "color": "#efe6d2"}
}

func seatProgress(seat CreationSeat) ui.Node {
	border := "rgba(217,164,65,.48)"
	statusColor := "#a89f8c"
	if seat.Ready {
		border, statusColor = "#3aa39a", "#3aa39a"
	}
	name := seat.Name
	if name == "" {
		name = "Seat " + strconv.Itoa(int(seat.Number))
	}
	species := seat.Species
	if species == "" {
		species = "—"
	}
	gender := seat.Gender
	if gender == "" {
		gender = "—"
	}
	children := []ui.Node{
		html.P(html.Props{Style: map[string]string{"margin": "0 0 .55rem", "color": "#a89f8c", "font-size": "clamp(.85rem,1.1vw,1.15rem)", "font-weight": "700", "letter-spacing": ".14em"}}, ui.Text("PLAYER "+strconv.Itoa(int(seat.Number)))),
		html.H2(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia,serif", "font-size": "clamp(1.8rem,3.4vw,4rem)"}}, ui.Text(name)),
		html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "1fr 1fr", "gap": ".8rem", "margin-top": "1.5rem", "color": "#a89f8c", "font-size": "clamp(1rem,1.6vw,1.7rem)"}},
			html.P(html.Props{Style: map[string]string{"margin": "0"}}, html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#d9a441", "font-size": ".65em", "letter-spacing": ".12em", "text-transform": "uppercase"}}, ui.Text("Species")), ui.Text(species)),
			html.P(html.Props{Style: map[string]string{"margin": "0"}}, html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#d9a441", "font-size": ".65em", "letter-spacing": ".12em", "text-transform": "uppercase"}}, ui.Text("Gender")), ui.Text(gender)),
		),
	}
	if seat.Class != "" {
		children = append(children, html.P(html.Props{Style: map[string]string{"margin": "1.4rem 0 0", "color": "#efe6d2", "font-size": "clamp(1.2rem,2vw,2rem)"}}, ui.Text("Rolled class: "+seat.Class)))
	}
	if seat.PortraitURL != "" {
		children = append(children, html.Img(html.Props{Src: seat.PortraitURL, Alt: name, Style: map[string]string{"display": "block", "height": "clamp(7rem,18vh,13rem)", "max-width": "100%", "margin": "1rem auto 0", "object-fit": "contain"}}))
	}
	children = append(children, html.P(html.Props{Style: map[string]string{"margin": "1.2rem 0 0", "color": statusColor, "font-size": "clamp(1.05rem,1.6vw,1.7rem)", "font-weight": "700"}, Aria: map[string]string{"live": "polite"}}, ui.Text(seat.Status)))
	return html.Div(html.Props{Class: "df-dm-creation-seat", Role: "listitem", Style: map[string]string{"min-height": "17rem", "padding": "clamp(1rem,2.4vw,2.5rem)", "border": "1px solid " + border, "border-radius": "14px", "background": "rgba(15,17,23,.76)", "box-shadow": "inset 0 0 28px rgba(0,0,0,.24)"}}, children...)
}
