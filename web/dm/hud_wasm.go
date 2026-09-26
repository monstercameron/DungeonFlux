//go:build js && wasm

package dm

import (
	"strconv"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ExplorationHUDComponent renders the TV exploration overlay from a snapshot.
func ExplorationHUDComponent(state *dungeonfluxv1.ScreenState) router.Component {
	model := HUDModelFromState(state)
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{Class: "df-dm-exploration-hud", Role: "region", Aria: map[string]string{"label": "Exploration HUD"}, Hidden: !model.Visible, Style: hudStyle()},
			hudParty(model.Party),
			hudObjective(model),
			hudActions(model),
		)
	}
}

func hudStyle() map[string]string {
	return map[string]string{
		"position": "relative", "width": "100%", "height": "100%", "color": "#efe6d2",
		"font-family": "Arial, sans-serif", "pointer-events": "none",
		"text-shadow": "0 2px 4px rgba(0,0,0,.72)",
	}
}

func hudParty(party []HUDPartyMember) ui.Node {
	children := make([]ui.Node, 0, len(party))
	for _, member := range party {
		children = append(children, hudPartyCard(member))
	}
	return html.Div(html.Props{Class: "df-dm-hud-party", Role: "list", Aria: map[string]string{"label": "Party"}, Style: map[string]string{
		"position": "absolute", "left": "1.5%", "top": "11%", "width": "23%", "display": "grid", "gap": ".8rem",
	}}, children...)
}

func hudPartyCard(member HUDPartyMember) ui.Node {
	border := "rgba(217,164,65,.42)"
	if member.Spotlight {
		border = "#d9a441"
	}
	portrait := hudPortrait(member)
	crest := hudCrest(member)
	copy := html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1"}},
		html.Strong(html.Props{Style: map[string]string{"display": "block", "overflow": "hidden", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.45vw, 1.75rem)", "line-height": "1.05", "text-overflow": "ellipsis", "white-space": "nowrap"}}, ui.Text(nameOrSeat(member))),
		html.Span(html.Props{Style: map[string]string{"display": "block", "margin-top": ".28rem", "color": "#a89f8c", "font-size": "clamp(.72rem, .9vw, 1.05rem)", "letter-spacing": ".08em", "text-transform": "uppercase"}}, ui.Text(classOrHero(member))),
		hudHP(member),
	)
	return html.Div(html.Props{Class: "df-dm-hud-party-card", Role: "listitem", Style: map[string]string{
		"display": "flex", "align-items": "center", "gap": ".65rem", "min-height": "clamp(4.3rem, 8vh, 6.5rem)",
		"padding": ".45rem .65rem .45rem .45rem", "border": "1px solid " + border, "border-radius": "10px",
		"background": "linear-gradient(90deg, rgba(10,12,17,.96), rgba(20,23,30,.78))", "box-shadow": spotlightShadow(member.Spotlight),
	}}, portrait, copy, crest)
}

func nameOrSeat(member HUDPartyMember) string {
	if member.Name != "" {
		return member.Name
	}
	return "Player " + strconv.Itoa(int(member.Number))
}

func classOrHero(member HUDPartyMember) string {
	if member.Class != "" {
		return member.Class
	}
	return "Hero"
}

func hudPortrait(member HUDPartyMember) ui.Node {
	style := map[string]string{"width": "clamp(3rem, 5vw, 5.4rem)", "height": "clamp(3rem, 5vw, 5.4rem)", "flex": "0 0 auto", "object-fit": "cover", "border": "1px solid rgba(217,164,65,.7)", "border-radius": "7px"}
	if member.PortraitURL != "" {
		return html.Img(html.Props{Src: member.PortraitURL, Alt: nameOrSeat(member), Style: style})
	}
	style["background"] = "radial-gradient(circle at 50% 25%, #4d5964, #111722 68%)"
	return html.Div(html.Props{Aria: map[string]string{"label": nameOrSeat(member) + " portrait placeholder"}, Style: style})
}

func hudCrest(member HUDPartyMember) ui.Node {
	crestURL := ArtURL(member.CrestArt)
	style := map[string]string{"width": "clamp(2rem, 3.2vw, 3.6rem)", "height": "clamp(2rem, 3.2vw, 3.6rem)", "flex": "0 0 auto", "object-fit": "contain", "opacity": ".94"}
	if crestURL != "" {
		return html.Img(html.Props{Src: crestURL, Alt: classOrHero(member) + " crest", Style: style})
	}
	style["border"] = "1px solid rgba(217,164,65,.75)"
	style["border-radius"] = "50%"
	style["background"] = "radial-gradient(circle, rgba(217,164,65,.4), rgba(15,17,23,.7) 62%)"
	return html.Div(html.Props{Aria: map[string]string{"label": classOrHero(member) + " crest placeholder"}, Style: style})
}

func hudHP(member HUDPartyMember) ui.Node {
	if !member.HPKnown {
		return html.Span(html.Props{Style: map[string]string{"display": "block", "margin-top": ".3rem", "color": "#a89f8c", "font-size": "clamp(.65rem, .8vw, .9rem)"}}, ui.Text("HP unavailable"))
	}
	return html.Div(html.Props{Style: map[string]string{"margin-top": ".35rem"}},
		html.Span(html.Props{Style: map[string]string{"display": "block", "color": "#a89f8c", "font-size": "clamp(.65rem, .8vw, .9rem)"}}, ui.Text("HP "+strconv.Itoa(int(member.HP))+"/"+strconv.Itoa(int(member.HPMax)))),
		html.Div(html.Props{Aria: map[string]string{"label": "Hit points"}, Style: map[string]string{"height": ".32rem", "margin-top": ".2rem", "overflow": "hidden", "border-radius": "99px", "background": "rgba(239,230,210,.2)"}},
			html.Div(html.Props{Style: map[string]string{"width": strconv.Itoa(member.HPPercent) + "%", "height": "100%", "background": hpColor(member.HPPercent), "border-radius": "inherit"}}),
		),
	)
}

func spotlightShadow(spotlight bool) string {
	if spotlight {
		return "0 0 24px rgba(217,164,65,.34), inset 0 0 18px rgba(217,164,65,.08)"
	}
	return "inset 0 0 18px rgba(0,0,0,.25)"
}

func hpColor(percent int) string {
	if percent <= 25 {
		return "#b3372f"
	}
	if percent <= 50 {
		return "#d28c39"
	}
	return "#3aa39a"
}

func hudObjective(model HUDModel) ui.Node {
	content := []ui.Node{html.Strong(html.Props{Style: map[string]string{"display": "block", "color": "#d9a441", "font-family": "Georgia, serif", "font-size": "clamp(.85rem, 1.05vw, 1.25rem)", "letter-spacing": ".06em", "text-transform": "uppercase"}}, ui.Text("Current objective"))}
	if model.ObjectiveVisible {
		content = append(content, html.P(html.Props{Style: map[string]string{"margin": ".7rem 0 0", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(1rem, 1.45vw, 1.8rem)", "line-height": "1.2"}}, ui.Text(model.Objective)))
	}
	return html.Div(html.Props{Class: "df-dm-hud-objective", Hidden: !model.ObjectiveVisible, Style: map[string]string{
		"position": "absolute", "top": "11%", "right": "2.5%", "width": "clamp(15rem, 24vw, 28rem)", "box-sizing": "border-box",
		"padding": "1rem 1.15rem 1.2rem", "border": "1px solid rgba(217,164,65,.7)", "border-radius": "10px",
		"background": "linear-gradient(135deg, rgba(10,12,17,.96), rgba(25,27,33,.86))", "box-shadow": "0 12px 28px rgba(0,0,0,.42), inset 0 0 18px rgba(217,164,65,.05)",
	}}, content...)
}

func hudActions(model HUDModel) ui.Node {
	items := make([]ui.Node, 0, len(model.Actions))
	for index, action := range model.Actions {
		items = append(items, html.Div(html.Props{Class: "df-dm-hud-action", Role: "listitem", Style: actionStyle(action.Enabled)},
			html.Span(html.Props{Style: map[string]string{"display": "block", "color": "#d9a441", "font-size": "clamp(.62rem, .75vw, .9rem)", "letter-spacing": ".16em"}}, ui.Text(strconv.Itoa(index+1))),
			html.Strong(html.Props{Style: map[string]string{"display": "block", "margin-top": ".18rem", "color": "#efe6d2", "font-family": "Georgia, serif", "font-size": "clamp(.85rem, 1.2vw, 1.35rem)"}}, ui.Text(action.Label)),
			html.Small(html.Props{Hidden: action.Enabled, Style: map[string]string{"display": "block", "margin-top": ".25rem", "color": "#a89f8c", "font-size": "clamp(.6rem, .7vw, .8rem)"}}, ui.Text(action.Reason)),
		))
	}
	return html.Div(html.Props{Class: "df-dm-hud-actions", Role: "list", Aria: map[string]string{"label": "Legal actions"}, Style: map[string]string{
		"position": "absolute", "left": "29%", "right": "14%", "bottom": "5.5%", "display": "flex", "justify-content": "center", "gap": "clamp(.5rem, 1vw, 1rem)",
	}}, items...)
}

func actionStyle(enabled bool) map[string]string {
	background := "rgba(13,16,22,.88)"
	border := "rgba(217,164,65,.52)"
	if !enabled {
		background = "rgba(13,16,22,.64)"
		border = "rgba(168,159,140,.3)"
	}
	return map[string]string{"min-width": "clamp(10rem, 14vw, 17rem)", "padding": ".7rem 1rem .8rem", "border": "1px solid " + border, "border-radius": "9px", "background": background, "box-shadow": "0 8px 20px rgba(0,0,0,.38)", "text-align": "center"}
}
