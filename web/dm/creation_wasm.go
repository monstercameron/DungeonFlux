//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CreationComponent renders the fixed-canvas character-creation tableau.
func CreationComponent(model CreationModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		featured := featuredCreationSeat(model)
		style := creationStyle()
		style["position"], style["left"], style["top"] = "absolute", "0", "0"
		style["width"], style["height"] = "1920px", "1080px"
		return html.Section(html.Props{Class: "df-dm-creation", Role: "region", Aria: map[string]string{"label": "Character creation"}, Style: style},
			creationHeader(model), creationPortraitPanel(featured), creationBuildPanel(featured), creationPhonePanel(featured), creationSeatStrip(model.Seats, featured.Number), creationLockup(featured),
		)
	}
}

func creationStyle() map[string]string {
	background := "linear-gradient(180deg,rgba(5,9,15,.24),rgba(5,8,14,.78)),linear-gradient(90deg,rgba(5,8,14,.86),rgba(5,8,14,.14) 53%,rgba(5,8,14,.84))"
	if art := ArtURL("ui/title_bg_wide"); art != "" {
		background += ",url('" + art + "')"
	}
	return map[string]string{"position": "relative", "width": "100%", "height": "100%", "min-height": "100%", "overflow": "hidden", "background": background, "background-position": "center", "background-size": "cover", "color": "#efe6d2", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif"}
}

func creationHeader(model CreationModel) ui.Node {
	brand := html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "205px", "top": "24px", "width": "500px", "height": "120px", "overflow": "hidden", "z-index": "3"}}, html.Div(html.Props{Style: map[string]string{"width": "735px", "transform": "scale(.68)", "transform-origin": "top left"}}, TitlePlate(ArtURL("ui/logo_wordmark"), "DungeonFlux", "AI DUNGEON MASTER FOR FIFTH-EDITION FANTASY")))
	title := html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "510px", "top": "174px", "width": "900px", "z-index": "3", "text-align": "center", "text-shadow": "0 4px 16px #000"}}, html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cinzel, 'Cormorant Garamond', Georgia, serif", "font-size": "66px", "font-weight": "500", "line-height": "1.05", "color": "#efe6d2"}}, html.Text(T("en", "dm.create.title", nil))), html.Div(html.Props{Style: map[string]string{"height": "1px", "width": "520px", "margin": "12px auto 10px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}), html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "28px", "color": "#efe6d2"}}, html.Text(T("en", "dm.create.hint", nil))))
	return html.Header(html.Props{Class: "df-dm-creation-header"}, brand, title, html.Div(html.Props{Style: map[string]string{"position": "absolute", "right": "48px", "top": "42px", "width": "250px", "color": "#bdb3a1", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px", "line-height": "1.2", "text-align": "right"}}, html.Text(model.Prompt)))
}

func creationPanelStyle(left, top, width, height string) map[string]string {
	return map[string]string{"position": "absolute", "left": left, "top": top, "width": width, "height": height, "z-index": "2", "overflow": "hidden", "border": "1px solid rgba(217,164,65,.78)", "border-radius": "12px", "background": "linear-gradient(145deg,rgba(12,18,28,.9),rgba(6,10,16,.95))", "box-shadow": "0 16px 38px rgba(0,0,0,.52),inset 0 0 25px rgba(217,164,65,.05)", "color": "#efe6d2"}
}

func creationCornerLabel(value string) ui.Node {
	return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#e7c27a", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "15px", "font-weight": "700", "letter-spacing": ".16em", "text-align": "center", "text-transform": "uppercase"}}, html.Text(value))
}

func creationPortraitPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("230px", "324px", "500px", "600px")
	portrait := creationAssetURL(seat.PortraitURL)
	if portrait == "" && (seat.Name != "" || seat.Class != "" || seat.Species != "") {
		portrait = heroProxyArt(seat.Species, seat.Class, seat.Name+strconv.Itoa(int(seat.Number)))
	}
	image := html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "radial-gradient(circle at 50% 30%,rgba(48,73,102,.9),transparent 45%),linear-gradient(145deg,#111b2a,#0b0d13)"}}, html.Img(html.Props{Src: portrait, Alt: creationDisplayValue(seat.Name, "Hero portrait"), Hidden: portrait == "", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover", "object-position": "center top"}}))
	return html.Div(html.Props{Class: "df-dm-creation-portrait", Role: "img", Aria: map[string]string{"label": "Generated hero portrait"}, Style: style},
		image,
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "linear-gradient(180deg,transparent 32%,rgba(5,7,11,.1) 50%,rgba(5,7,11,.98) 100%)"}}),
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "24px", "right": "24px", "bottom": "22px", "padding": "16px 14px", "border": "1px solid rgba(217,164,65,.82)", "background": "rgba(8,11,16,.86)", "text-align": "center"}},
			html.H2(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cinzel, Georgia, serif", "font-size": "34px", "font-weight": "500"}}, html.Text(creationDisplayValue(seat.Name, "Waiting for hero"))),
			html.P(html.Props{Style: map[string]string{"margin": "7px 0 0", "color": "#e7c27a", "font-size": "17px", "font-weight": "700", "letter-spacing": ".16em", "text-transform": "uppercase"}}, html.Text(creationDisplayValue(seat.Class, "Class pending"))),
		),
	)
}

func creationBuildPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("770px", "324px", "570px", "520px")
	return html.Div(html.Props{Class: "df-dm-creation-build", Style: style}, creationCornerLabel("ENGINE GENERATED STATS"), html.P(html.Props{Style: map[string]string{"margin": "12px 0 20px", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px", "text-align": "center", "color": "#c8bda8"}}, html.Text(T("en", "dm.create.stats", nil))), creationStatGrid(), html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "12px", "margin-top": "22px", "color": "#d9a441", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "21px"}}, html.Div(html.Props{Style: map[string]string{"height": "1px", "flex": "1", "background": "linear-gradient(90deg,transparent,#d9a441)"}}), html.Text(creationDisplayValue(seat.Status, "Rolls complete")), html.Div(html.Props{Style: map[string]string{"height": "1px", "flex": "1", "background": "linear-gradient(90deg,#d9a441,transparent)"}})))
}

func creationStatGrid() ui.Node {
	stats := [][2]string{{"STR", "16"}, {"DEX", "14"}, {"CON", "14"}, {"INT", "10"}, {"WIS", "12"}, {"CHA", "10"}}
	items := make([]ui.Node, 0, len(stats))
	for _, stat := range stats {
		items = append(items, html.Div(html.Props{Style: map[string]string{"height": "126px", "padding": "13px 8px", "border": "1px solid rgba(184,137,58,.78)", "border-radius": "7px", "background": "rgba(10,17,27,.76)", "text-align": "center"}}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel, Georgia, serif", "font-size": "20px", "letter-spacing": ".08em"}}, html.Text(stat[0])), html.Div(html.Props{Style: map[string]string{"margin-top": "8px", "font-family": "Cinzel, Georgia, serif", "font-size": "34px", "line-height": "1"}}, html.Text(stat[1])), html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "7px", "color": "#c8bda8", "font-size": "18px"}}, html.Text(statBonus(stat[1])))))
	}
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(3,1fr)", "gap": "12px"}}, items...)
}

func statBonus(value string) string {
	if value == "16" {
		return "+3"
	}
	if value == "14" {
		return "+2"
	}
	if value == "12" {
		return "+1"
	}
	return "+0"
}

func creationPhonePanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("1400px", "224px", "470px", "724px")
	return html.Div(html.Props{Class: "df-dm-creation-phone", Style: style}, html.Div(html.Props{Style: map[string]string{"padding": "25px 24px"}}, creationCornerLabel("YOUR PHONE CONTROLS THE HERO"), html.Div(html.Props{Style: map[string]string{"height": "1px", "margin": "15px 0 20px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}), html.Div(html.Props{Style: map[string]string{"color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "20px", "letter-spacing": ".1em"}}, html.Text(T("en", "dm.create.step_gender", nil))), creationChoiceRow("♂", creationDisplayValue(seat.Gender, "Male"), "female" == strings.ToLower(seat.Gender)), creationChoiceRow("♀", "Female", strings.EqualFold(seat.Gender, "female")), creationChoiceRow("○", "Nonbinary", strings.EqualFold(seat.Gender, "nonbinary")), html.Div(html.Props{Style: map[string]string{"margin-top": "24px", "color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "20px", "letter-spacing": ".1em"}}, html.Text(T("en", "dm.create.step_race", nil))), creationChoiceRow("◐", creationDisplayValue(seat.Species, "Human"), true), creationChoiceRow("◈", "Elf", strings.EqualFold(seat.Species, "elf")), creationChoiceRow("◆", "Dwarf", strings.EqualFold(seat.Species, "dwarf")), html.Div(html.Props{Style: map[string]string{"margin-top": "24px", "color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "20px", "letter-spacing": ".1em"}}, html.Text(T("en", "dm.create.step_class", nil))), creationChoiceRow("⚔", creationDisplayValue(seat.Class, "Fighter"), true), creationChoiceRow("♠", "Rogue", strings.EqualFold(seat.Class, "rogue")), creationChoiceRow("✧", "Wizard", strings.EqualFold(seat.Class, "wizard")), html.Div(html.Props{Style: map[string]string{"margin-top": "20px", "height": "64px", "display": "grid", "place-items": "center", "border": "1px solid #e7c27a", "border-radius": "8px", "background": "linear-gradient(180deg,#d9a441,#8c541d)", "box-shadow": "0 0 22px rgba(217,164,65,.28)", "color": "#211a12", "font-family": "Cinzel, Georgia, serif", "font-size": "23px"}}, html.Text(T("en", "dm.create.generate", nil))), html.P(html.Props{Style: map[string]string{"margin": "18px 0 0", "color": "#a89f8c", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px", "text-align": "center"}}, html.Text(T("en", "dm.create.choices", nil)))))
}

func creationChoiceRow(icon, value string, selected bool) ui.Node {
	border, background := "rgba(184,137,58,.55)", "rgba(12,23,34,.8)"
	shadow := "none"
	if selected {
		border, background = "#e7c27a", "rgba(38,65,83,.94)"
		shadow = "0 0 15px rgba(217,164,65,.25)"
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "12px", "height": "58px", "margin-top": "7px", "padding": "6px 12px", "border": "1px solid " + border, "border-radius": "6px", "background": background, "box-shadow": shadow}}, html.Span(html.Props{Style: map[string]string{"width": "36px", "color": "#e7c27a", "font-size": "25px", "text-align": "center"}}, html.Text(icon)), html.Span(html.Props{Style: map[string]string{"font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px", "font-weight": "600"}}, html.Text(value)))
}

func creationSeatStrip(seats [2]CreationSeat, featured int32) ui.Node {
	items := make([]ui.Node, 0, len(seats))
	for _, seat := range seats {
		if seat.Number == featured {
			continue
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "10px", "padding": "9px 14px", "border": "1px solid rgba(217,164,65,.46)", "border-radius": "7px", "background": "rgba(8,11,16,.86)", "color": "#efe6d2"}}, html.Span(html.Props{Style: map[string]string{"color": "#d9a441", "font-size": "18px"}}, html.Text(T("en", "dm.create.glyph", nil))), html.Div(html.Props{}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel, Georgia, serif", "font-size": "18px"}}, html.Text(creationDisplayValue(seat.Name, "Seat "+strconv.Itoa(int(seat.Number))))), html.Small(html.Props{Style: map[string]string{"color": "#a89f8c", "font-size": "15px"}}, html.Text(creationDisplayValue(seat.Status, "Waiting for player"))))))
	}
	return html.Div(html.Props{Class: "df-dm-creation-seat-strip", Style: map[string]string{"position": "absolute", "left": "230px", "bottom": "38px", "z-index": "4", "display": "flex", "gap": "10px"}}, items...)
}

func creationLockup(seat CreationSeat) ui.Node {
	label := "Lock In Character"
	if !seat.Ready {
		label = "Awaiting Roll"
	}
	return html.Div(html.Props{Class: "df-dm-creation-lockup", Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"position": "absolute", "left": "790px", "bottom": "38px", "width": "530px", "height": "70px", "z-index": "4", "display": "grid", "place-items": "center", "border": "1px solid #e7c27a", "border-radius": "9px", "background": "linear-gradient(180deg,#d9a441,#8c541d)", "box-shadow": "0 0 20px rgba(217,164,65,.3)", "color": "#211a12", "font-family": "Cinzel, Georgia, serif", "font-size": "28px"}}, html.Text("✦  "+label))
}
