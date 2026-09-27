//go:build js && wasm

package dm

import (
	"strconv"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ExplorationHUDComponent renders the exploration overlay on the fixed TV
// canvas. Geometry is intentionally expressed in design-canvas pixels so the
// 1920x1080 composition remains stable at 16:9 and ultrawide viewports.
func ExplorationHUDComponent(state *dungeonfluxv1.ScreenState) router.Component {
	model := HUDModelFromState(state)
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{
			Class: "df-dm-exploration-hud", Role: "region",
			Aria: map[string]string{"label": "Exploration HUD"}, Hidden: !model.Visible,
			Style: hudRootStyle(),
		}, hudTitlePlate(), hudParty(model.Party), hudLocation(model), hudObjective(model),
			hudNarration(model), hudActions(model), hudMinimap(model), hudFooter(model))
	}
}

func hudRootStyle() map[string]string {
	return map[string]string{
		"position": "absolute", "inset": "0", "width": "1920px", "height": "1080px",
		"color": "#efe6d2", "font-family": "Inter, Arial, sans-serif", "pointer-events": "none",
		"text-shadow": "0 2px 5px rgba(0,0,0,.78)", "z-index": "60",
	}
}

func hudTitlePlate() ui.Node {
	return html.Div(html.Props{Class: "df-dm-hud-brand", Style: map[string]string{"position": "absolute", "left": "34px", "top": "22px", "z-index": "5"}}, CornerBrand())
}

func hudParty(party []HUDPartyMember) ui.Node {
	children := make([]ui.Node, 0, len(party))
	for _, member := range party {
		children = append(children, hudPartyCard(member))
	}
	return html.Div(html.Props{Class: "df-dm-hud-party", Role: "list", Aria: map[string]string{"label": "Party"}, Style: map[string]string{
		"position": "absolute", "left": "30px", "top": "135px", "width": "300px", "display": "grid", "gap": "10px",
	}}, children...)
}

func hudPartyCard(member HUDPartyMember) ui.Node {
	border := "rgba(184,137,58,.72)"
	shadow := "inset 0 0 18px rgba(0,0,0,.32)"
	if member.Spotlight {
		border = "#e7c27a"
		shadow = "0 0 24px rgba(217,164,65,.35), inset 0 0 18px rgba(217,164,65,.1)"
	}
	portraitStyle := map[string]string{
		"width": "92px", "height": "92px", "flex": "0 0 92px", "object-fit": "cover", "border": "2px solid " + border,
		"border-radius": "7px", "background": "radial-gradient(circle at 50% 25%,#4d5964,#111722 68%)",
	}
	var portrait ui.Node
	resolved := artSrc(member.PortraitURL)
	if resolved == "" {
		resolved = heroProxyArt("", member.Class, nameOrSeat(member))
	}
	if resolved != "" {
		portrait = html.Img(html.Props{Src: resolved, Alt: nameOrSeat(member), Style: portraitStyle})
	} else {
		portrait = html.Div(html.Props{Aria: map[string]string{"label": nameOrSeat(member) + " portrait placeholder"}, Style: portraitStyle},
			html.Span(html.Props{Style: map[string]string{"display": "block", "padding-top": "25px", "color": "#d9a441", "font-size": "28px", "text-align": "center"}}, ui.Text(T("en", "dm.glyph.star", nil))))
	}
	return html.Article(html.Props{Class: "df-dm-hud-party-card", Role: "listitem", Style: map[string]string{
		"display": "flex", "align-items": "center", "gap": "12px", "height": "100px", "padding": "4px", "border": "1px solid " + border,
		"border-radius": "10px", "background": "linear-gradient(90deg,rgba(10,12,17,.96),rgba(20,23,30,.82))", "box-shadow": shadow,
	}}, portrait, hudPartyCopy(member))
}

func hudPartyCopy(member HUDPartyMember) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1", "padding-right": "6px"}},
		html.Strong(html.Props{Style: map[string]string{
			"display": "block", "overflow": "hidden", "color": "#efe6d2", "font-family": "Cinzel, Georgia, serif", "font-size": "24px", "line-height": "1.05", "text-overflow": "ellipsis", "white-space": "nowrap",
		}}, ui.Text(nameOrSeat(member))),
		html.Span(html.Props{Style: map[string]string{
			"display": "block", "margin-top": "4px", "overflow": "hidden", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px", "text-overflow": "ellipsis", "white-space": "nowrap",
		}}, ui.Text(classOrHero(member))), hudHP(member))
}

func hudHP(member HUDPartyMember) ui.Node {
	if !member.HPKnown {
		return html.Span(html.Props{Style: map[string]string{"display": "block", "margin-top": "5px", "color": "#a89f8c", "font-size": "14px"}}, ui.Text(T("en", "dm.hp_unavailable", nil)))
	}
	return html.Div(html.Props{Style: map[string]string{"margin-top": "5px"}},
		html.Span(html.Props{Style: map[string]string{"display": "block", "color": "#a89f8c", "font-size": "14px"}}, ui.Text("HP "+strconv.Itoa(int(member.HP))+"/"+strconv.Itoa(int(member.HPMax)))),
		html.Div(html.Props{Aria: map[string]string{"label": "Hit points"}, Style: map[string]string{"width": "120px", "height": "7px", "margin-top": "3px", "overflow": "hidden", "border-radius": "99px", "background": "rgba(239,230,210,.2)"}},
			html.Div(html.Props{Style: map[string]string{"width": strconv.Itoa(member.HPPercent) + "%", "height": "100%", "background": hpColor(member.HPPercent), "border-radius": "inherit"}})),
	)
}

func hudLocation(model HUDModel) ui.Node {
	return html.Div(html.Props{Class: "df-location-title", Style: map[string]string{
		"position": "absolute", "right": "30px", "top": "25px", "width": "290px", "text-align": "right",
	}}, LocationTitle(model.Location, model.Act))
}

func hudObjective(model HUDModel) ui.Node {
	if !model.ObjectiveVisible {
		return html.Section(html.Props{Hidden: true})
	}
	items := []ui.Node{html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "10px", "align-items": "flex-start", "margin-top": "2px"}},
		html.Span(html.Props{Style: map[string]string{"color": "#d9a441", "font-size": "22px", "line-height": "1"}}, ui.Text(T("en", "dm.glyph.objective", nil))),
		html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "24px", "line-height": "1.12"}}, ui.Text(model.Objective)))}
	for _, item := range model.Checklist {
		color := "#a89f8c"
		mark := "○"
		if item.Done {
			color, mark = "#3aa39a", "●"
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "10px", "align-items": "center", "margin-top": "11px", "color": color, "font-size": "18px"}},
			html.Span(html.Props{Style: map[string]string{"font-size": "20px"}}, ui.Text(mark)), html.Span(html.Props{}, ui.Text(item.Label))))
	}
	return hudPanel("Current Objective", map[string]string{
		"position": "absolute", "right": "30px", "top": "125px", "width": "330px", "min-height": "275px", "padding": "22px 22px 18px",
	}, items...)
}

func hudNarration(model HUDModel) ui.Node {
	if model.NarrationText == "" {
		return html.Section(html.Props{Hidden: true})
	}
	portraitURL := artSrc("ui/dm_speaker")
	portraitStyle := map[string]string{"width": "118px", "height": "118px", "flex": "0 0 118px", "border": "2px solid #d9a441", "border-radius": "50%", "object-fit": "cover", "background": "radial-gradient(circle at 50% 35%,#293241,#07090d 70%)"}
	var portrait ui.Node = html.Div(html.Props{Style: portraitStyle}, html.Span(html.Props{Style: map[string]string{"display": "block", "padding-top": "36px", "color": "#d9a441", "font-size": "40px", "text-align": "center"}}, ui.Text(T("en", "dm.glyph.star", nil))))
	if portraitURL != "" {
		portrait = html.Img(html.Props{Src: portraitURL, Alt: T("en", "dm.speaker_alt", nil), Style: portraitStyle})
	}
	return hudPanel("", map[string]string{
		"position": "absolute", "left": "25px", "bottom": "90px", "width": "745px", "height": "165px", "padding": "20px 24px",
		"display": "flex", "align-items": "center", "gap": "20px",
	}, portrait, html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1"}},
		html.Strong(html.Props{Style: map[string]string{"display": "block", "color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "25px", "font-weight": "500"}}, ui.Text(model.NarrationSpeaker)),
		html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": "8px 0 10px", "background": "linear-gradient(90deg,#d9a441,transparent)"}}),
		html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "24px", "font-style": "italic", "line-height": "1.2"}}, ui.Text(model.NarrationText)),
	))
}

func hudActions(model HUDModel) ui.Node {
	items := make([]ui.Node, 0, len(model.Actions))
	for _, action := range model.Actions {
		items = append(items, ActionButton(ActionButtonModel{
			Icon: action.Icon, Label: action.Label, Hotkey: action.Hotkey, Primary: action.Primary,
			Enabled: action.Enabled, Reason: action.Reason,
		}))
	}
	return html.Div(html.Props{Class: "df-dm-hud-actions", Role: "list", Aria: map[string]string{"label": "Legal actions"}, Style: map[string]string{
		"position": "absolute", "left": "945px", "bottom": "90px", "width": "755px", "height": "135px", "display": "flex", "align-items": "flex-start", "justify-content": "center", "gap": "18px",
	}}, items...)
}

func hudMinimap(model HUDModel) ui.Node {
	style := map[string]string{"position": "absolute", "right": "30px", "top": "590px", "width": "240px", "height": "240px", "overflow": "hidden", "border": "2px solid #b8893a", "border-radius": "50%", "background-image": "radial-gradient(circle,#263746,#0b1018 72%)", "box-shadow": "0 0 0 7px rgba(12,18,28,.76),0 10px 25px rgba(0,0,0,.55)"}
	if minimapURL := artSrc(model.MinimapURL); minimapURL != "" {
		style["background-image"] = "linear-gradient(rgba(7,12,18,.2),rgba(7,12,18,.45)),url('" + minimapURL + "')"
		style["background-size"] = "cover"
		style["background-position"] = "center"
	}
	return html.Div(html.Props{Class: "df-dm-hud-minimap", Role: "img", Aria: map[string]string{"label": "Exploration minimap"}, Style: style},
		html.Span(html.Props{Style: map[string]string{"position": "absolute", "left": "50%", "top": "50%", "color": "#e7c27a", "font-size": "28px", "transform": "translate(-50%,-50%)"}}, ui.Text(T("en", "dm.glyph.minimap", nil))))
}

func hudFooter(model HUDModel) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"position": "absolute", "right": "30px", "bottom": "28px", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "17px", "letter-spacing": ".04em"}}, ui.Text("Session: "+model.Location))
}

func hudPanel(title string, style map[string]string, children ...ui.Node) ui.Node {
	content := make([]ui.Node, 0, len(children)+1)
	if title != "" {
		content = append(content, html.H2(html.Props{Class: "df-ornate-panel-title", Style: map[string]string{"margin": "0 0 18px", "font-size": "19px"}}, ui.Text(title)))
	}
	content = append(content, children...)
	return html.Section(html.Props{Class: "df-ornate-panel", Role: "region", Style: style}, content...)
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

func hpColor(percent int) string {
	if percent <= 25 {
		return "#b3372f"
	}
	if percent <= 50 {
		return "#d28c39"
	}
	return "#3aa39a"
}
