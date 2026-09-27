//go:build js && wasm

package dm

import (
	"fmt"
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
			creationHeader(), creationPortraitPanel(featured), creationBuildPanel(featured), creationPhonePanel(featured), creationSeatStrip(model.Seats, featured.Number), creationLockup(featured),
		)
	}
}

func creationStyle() map[string]string {
	// One continuous cover background shared with the lobby (ui/title_bg_wide):
	// do not add a second, narrower background layer here, or the seam
	// between it and the surrounding page reappears at the layer's edge.
	background := "linear-gradient(180deg,rgba(5,9,15,.24),rgba(5,8,14,.78)),linear-gradient(90deg,rgba(5,8,14,.86),rgba(5,8,14,.14) 53%,rgba(5,8,14,.84))"
	if art := ArtURL("ui/title_bg_wide"); art != "" {
		background += ",url('" + art + "')"
	}
	return map[string]string{"position": "relative", "width": "100%", "height": "100%", "min-height": "100%", "overflow": "hidden", "background-image": background, "background-position": "center", "background-size": "cover", "color": "#efe6d2", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif"}
}

// creationHeader uses the same small corner wordmark band as every other TV
// screen (CornerBrand) instead of a scaled, clipped copy of the full title
// plate, so the wordmark never clips and never collides with the subtitle.
func creationHeader() ui.Node {
	brand := html.Div(html.Props{Class: "df-dm-creation-brand", Style: map[string]string{"position": "absolute", "left": "34px", "top": "22px", "z-index": "3"}}, CornerBrand())
	title := html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "0", "right": "0", "top": "30px", "z-index": "3", "text-align": "center", "text-shadow": "0 4px 16px #000"}},
		html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cinzel, 'Cormorant Garamond', Georgia, serif", "font-size": "50px", "font-weight": "500", "line-height": "1.05", "color": "#efe6d2"}}, html.Text(T("en", "dm.create.title", nil))),
		html.Div(html.Props{Style: map[string]string{"height": "1px", "width": "420px", "margin": "12px auto 8px", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}}),
		html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "22px", "color": "#c8bda8"}}, html.Text(T("en", "dm.create.hint", nil))),
	)
	return html.Header(html.Props{Class: "df-dm-creation-header"}, brand, title)
}

func creationPanelStyle(left, top, width, height string) map[string]string {
	return map[string]string{"position": "absolute", "left": left, "top": top, "width": width, "height": height, "z-index": "2", "overflow": "hidden", "border": "1px solid rgba(217,164,65,.78)", "border-radius": "12px", "background": "linear-gradient(145deg,rgba(12,18,28,.9),rgba(6,10,16,.95))", "box-shadow": "0 16px 38px rgba(0,0,0,.52),inset 0 0 25px rgba(217,164,65,.05)", "color": "#efe6d2"}
}

func creationCornerLabel(value string) ui.Node {
	return html.P(html.Props{Style: map[string]string{"margin": "0", "color": "#e7c27a", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "15px", "font-weight": "700", "letter-spacing": ".16em", "text-align": "center", "text-transform": "uppercase"}}, html.Text(value))
}

func creationGoldRule() ui.Node {
	return html.Div(html.Props{Style: map[string]string{"height": "1px", "flex": "none", "background": "linear-gradient(90deg,transparent,#d9a441,transparent)"}})
}

// creationPortraitPanel shows the seat's real generated portrait once it
// exists, or a gendered species stand-in, bleeding to the full height of an
// ornate double-bordered frame instead of an empty navy gradient.
func creationPortraitPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("230px", "324px", "500px", "600px")
	style["border"] = "3px solid rgba(217,164,65,.85)"
	style["box-shadow"] = "0 18px 44px rgba(0,0,0,.6), inset 0 0 0 7px rgba(7,11,17,.92), inset 0 0 0 8px rgba(217,164,65,.4)"
	portrait := creationHeroStandIn(seat)
	image := html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "radial-gradient(circle at 50% 30%,rgba(48,73,102,.9),transparent 45%),linear-gradient(145deg,#111b2a,#0b0d13)"}}, html.Img(html.Props{Src: portrait, Alt: creationDisplayValue(seat.Name, "Hero portrait"), Hidden: portrait == "", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover", "object-position": "center 12%"}}))
	nameplateChildren := make([]ui.Node, 0, 3)
	if seat.ClassCrestURL != "" {
		nameplateChildren = append(nameplateChildren, html.Img(html.Props{Src: seat.ClassCrestURL, Alt: "", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"display": "block", "width": "48px", "height": "48px", "max-width": "48px", "max-height": "48px", "margin": "0 auto 8px", "object-fit": "contain"}}))
	}
	nameplateChildren = append(nameplateChildren,
		html.H2(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cinzel, Georgia, serif", "font-size": "32px", "font-weight": "500", "overflow": "hidden", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(creationDisplayValue(seat.Name, "Waiting for hero"))),
		html.P(html.Props{Style: map[string]string{"margin": "7px 0 0", "color": "#e7c27a", "font-size": "17px", "font-weight": "700", "letter-spacing": ".14em", "text-transform": "uppercase", "overflow": "hidden", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(creationDisplayValue(seat.Class, "Class pending"))),
	)
	return html.Div(html.Props{Class: "df-dm-creation-portrait", Role: "img", Aria: map[string]string{"label": "Generated hero portrait"}, Style: style},
		image,
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "linear-gradient(180deg,transparent 32%,rgba(5,7,11,.1) 50%,rgba(5,7,11,.98) 100%)"}}),
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "20px", "right": "20px", "bottom": "20px", "max-width": "460px", "box-sizing": "border-box", "padding": "16px 14px", "border": "1px solid rgba(217,164,65,.82)", "background": "rgba(8,11,16,.86)", "text-align": "center"}}, nameplateChildren...),
	)
}

// creationBuildPanel shows the seat's real roll: six ability scores, HP/AC,
// and the picks that produced them. Placeholders ("—") show until the seat
// has a roll, so the panel never invents a fixed sample roll.
func creationBuildPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("770px", "324px", "570px", "520px")
	style["display"], style["flex-direction"] = "flex", "column"
	style["padding"], style["gap"] = "22px 26px 20px", "12px"
	header := html.Div(html.Props{}, creationCornerLabel("ENGINE GENERATED STATS"))
	rule := creationGoldRule()
	hint := html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px", "text-align": "center", "color": "#c8bda8"}}, html.Text(T("en", "dm.create.stats", nil)))
	grid := creationStatGrid(seat)
	secondary := creationSecondaryRow(seat)
	picks := creationPicksRow(seat)
	status := html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "12px", "margin-top": "auto", "color": "#d9a441", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px"}}, creationGoldRule(), html.Span(html.Props{Style: map[string]string{"white-space": "nowrap"}}, html.Text(creationDisplayValue(seat.Status, "Waiting for player"))), creationGoldRule())
	return html.Div(html.Props{Class: "df-dm-creation-build", Style: style}, header, rule, hint, grid, secondary, picks, status)
}

func creationStatGrid(seat CreationSeat) ui.Node {
	labels := [6]string{"STR", "DEX", "CON", "INT", "WIS", "CHA"}
	values := [6]int32{seat.Scores.STR, seat.Scores.DEX, seat.Scores.CON, seat.Scores.INT, seat.Scores.WIS, seat.Scores.CHA}
	items := make([]ui.Node, 0, 6)
	for index, label := range labels {
		valueText, bonusText := "—", "—"
		if seat.HasStats {
			valueText = strconv.Itoa(int(values[index]))
			bonusText = signedInt(abilityModifier(values[index]))
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"box-sizing": "border-box", "padding": "10px 6px", "border": "1px solid rgba(184,137,58,.78)", "border-radius": "7px", "background": "rgba(10,17,27,.76)", "text-align": "center"}},
			html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel, Georgia, serif", "font-size": "17px", "letter-spacing": ".08em"}}, html.Text(label)),
			html.Div(html.Props{Style: map[string]string{"margin-top": "5px", "font-family": "Cinzel, Georgia, serif", "font-size": "28px", "line-height": "1"}}, html.Text(valueText)),
			html.Small(html.Props{Style: map[string]string{"display": "block", "margin-top": "5px", "color": "#c8bda8", "font-size": "15px"}}, html.Text(bonusText)),
		))
	}
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(3,1fr)", "gap": "10px"}}, items...)
}

// creationSecondaryRow shows the seat's HP and AC beside the six scores.
func creationSecondaryRow(seat CreationSeat) ui.Node {
	hpText, acText := "—", "—"
	if seat.HasStats {
		hpText = strconv.Itoa(int(seat.HP)) + " / " + strconv.Itoa(int(seat.HPMax))
		acText = strconv.Itoa(int(seat.AC))
	}
	tile := func(label, value string) ui.Node {
		return html.Div(html.Props{Style: map[string]string{"flex": "1", "box-sizing": "border-box", "padding": "9px 10px", "border": "1px solid rgba(184,137,58,.78)", "border-radius": "7px", "background": "rgba(10,17,27,.76)", "text-align": "center"}},
			html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#c8bda8", "font-size": "13px", "letter-spacing": ".12em", "text-transform": "uppercase"}}, html.Text(label)),
			html.Div(html.Props{Style: map[string]string{"margin-top": "4px", "font-family": "Cinzel, Georgia, serif", "font-size": "22px"}}, html.Text(value)),
		)
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "10px"}}, tile("Hit Points", hpText), tile("Armor Class", acText))
}

// creationPicksRow shows the picks (species, gender, class) that produced
// the roll, title-cased, next to the numbers they produced.
func creationPicksRow(seat CreationSeat) ui.Node {
	pick := func(label, value string) ui.Node {
		return html.Div(html.Props{Style: map[string]string{"flex": "1", "min-width": "0", "text-align": "center"}},
			html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#a89f8c", "font-size": "12px", "letter-spacing": ".1em", "text-transform": "uppercase"}}, html.Text(label)),
			html.Strong(html.Props{Style: map[string]string{"display": "block", "margin-top": "3px", "overflow": "hidden", "text-overflow": "ellipsis", "white-space": "nowrap", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px", "color": "#efe6d2"}}, html.Text(value)),
		)
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "gap": "8px"}},
		pick("Species", creationDisplayValue(seat.Species, "—")),
		pick("Gender", creationDisplayValue(seat.Gender, "—")),
		pick("Class", creationDisplayValue(seat.Class, "—")),
	)
}

func signedInt(value int32) string {
	if value >= 0 {
		return "+" + strconv.Itoa(int(value))
	}
	return strconv.Itoa(int(value))
}

// pickTile is one selectable species or class option mirrored from the
// phone's own list (web/phone/create.go, web/phone/class.go) so the TV shows
// the same 9 species and 12 classes the phone offers.
type pickTile struct{ id, label, glyph string }

var creationSpeciesTiles = []pickTile{
	{"human", "Human", "◐"}, {"elf", "Elf", "◈"}, {"dwarf", "Dwarf", "◆"},
	{"halfling", "Halfling", "○"}, {"orc", "Orc", "▲"}, {"tiefling", "Tiefling", "☾"},
	{"dragonborn", "Dragonborn", "❖"}, {"gnome", "Gnome", "✥"}, {"goliath", "Goliath", "⬢"},
}

var creationGenderTiles = []pickTile{
	{"female", "Female", "♀"}, {"male", "Male", "♂"}, {"nonbinary", "Nonbinary", "○"},
}

var creationClassTiles = []pickTile{
	{"barbarian", "Barbarian", "⚔"}, {"bard", "Bard", "♫"}, {"cleric", "Cleric", "✚"},
	{"druid", "Druid", "❧"}, {"fighter", "Fighter", "◈"}, {"monk", "Monk", "☯"},
	{"paladin", "Paladin", "✦"}, {"ranger", "Ranger", "⌁"}, {"rogue", "Rogue", "◒"},
	{"sorcerer", "Sorcerer", "✧"}, {"warlock", "Warlock", "☽"}, {"wizard", "Wizard", "✺"},
}

// creationPhonePanel mirrors the phone's own picker as an icon-tile grid
// inside a phone-frame mock, with the seat's current picks highlighted, so
// the TV shows the real 9 species / 12 classes the phone offers instead of a
// truncated 3-item list.
func creationPhonePanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle("1400px", "224px", "470px", "724px")
	style["border-radius"] = "30px"
	style["display"], style["flex-direction"] = "flex", "column"
	style["padding"], style["gap"] = "20px 22px 18px", "10px"
	notch := html.Div(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": "64px", "height": "5px", "margin": "0 auto 4px", "border-radius": "3px", "background": "rgba(217,164,65,.45)"}})
	header := creationCornerLabel("YOUR PHONE CONTROLS THE HERO")
	sectionLabel := func(value string) ui.Node {
		return html.Div(html.Props{Style: map[string]string{"margin-top": "2px", "color": "#e7c27a", "font-family": "Cinzel, Georgia, serif", "font-size": "16px", "letter-spacing": ".1em"}}, html.Text(value))
	}
	return html.Div(html.Props{Class: "df-dm-creation-phone", Style: style},
		notch, header, creationGoldRule(),
		sectionLabel("GENDER"), creationPickGrid(creationGenderTiles, 3, seat.Gender, 46),
		sectionLabel("SPECIES"), creationPickGrid(creationSpeciesTiles, 3, seat.Species, 62),
		sectionLabel("CLASS"), creationPickGrid(creationClassTiles, 4, seat.Class, 56),
	)
}

func creationPickGrid(tiles []pickTile, columns int, current string, tileHeight int) ui.Node {
	items := make([]ui.Node, 0, len(tiles))
	for _, tile := range tiles {
		items = append(items, creationPickTile(tile, strings.EqualFold(tile.id, strings.TrimSpace(current)), tileHeight))
	}
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": fmt.Sprintf("repeat(%d,1fr)", columns), "gap": "7px"}}, items...)
}

func creationPickTile(tile pickTile, selected bool, height int) ui.Node {
	border, background, color, shadow := "rgba(184,137,58,.5)", "rgba(10,17,27,.7)", "#c8bda8", "none"
	if selected {
		border, background, color, shadow = "#e7c27a", "rgba(58,90,74,.55)", "#fff4dc", "0 0 12px rgba(217,164,65,.4)"
	}
	return html.Div(html.Props{Style: map[string]string{
		"box-sizing": "border-box", "height": strconv.Itoa(height) + "px", "display": "flex", "flex-direction": "column",
		"align-items": "center", "justify-content": "center", "gap": "3px", "border": "1px solid " + border,
		"border-radius": "7px", "background": background, "padding": "4px 2px", "box-shadow": shadow,
	}},
		html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"color": "#e7c27a", "font-size": "18px", "line-height": "1"}}, html.Text(tile.glyph)),
		html.Span(html.Props{Style: map[string]string{"color": color, "font-family": "Inter,ui-sans-serif,system-ui,sans-serif", "font-size": "11px", "font-weight": "600", "text-align": "center", "line-height": "1.1"}}, html.Text(tile.label)),
	)
}

func creationSeatStrip(seats [2]CreationSeat, featured int32) ui.Node {
	items := make([]ui.Node, 0, len(seats))
	for _, seat := range seats {
		if seat.Number == featured {
			continue
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "10px", "padding": "9px 14px", "border": "1px solid rgba(217,164,65,.46)", "border-radius": "7px", "background": "rgba(8,11,16,.86)", "color": "#efe6d2"}}, html.Span(html.Props{Style: map[string]string{"color": "#d9a441", "font-size": "18px"}}, html.Text(T("en", "dm.create.glyph", nil))), html.Div(html.Props{}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Cinzel, Georgia, serif", "font-size": "18px"}}, html.Text(creationDisplayValue(seat.Name, "Seat "+strconv.Itoa(int(seat.Number))))), html.Small(html.Props{Style: map[string]string{"color": "#a89f8c", "font-size": "15px"}}, html.Text(creationDisplayValue(seat.Status, "Waiting for player"))))))
	}
	return html.Div(html.Props{Class: "df-dm-creation-seat-strip", Style: map[string]string{"position": "absolute", "left": "230px", "bottom": "48px", "z-index": "4", "display": "flex", "gap": "10px"}}, items...)
}

// creationLockup shows the lock-in state: muted while the seat has not
// rolled (never active gold, which reads as "ready" before it is true), gold
// once the roll is real, and lifted clear of the canvas's bottom edge.
func creationLockup(seat CreationSeat) ui.Node {
	label := "Ready for adventure"
	style := map[string]string{"position": "absolute", "left": "790px", "bottom": "48px", "width": "530px", "height": "68px", "z-index": "4", "display": "grid", "place-items": "center", "font-family": "Cinzel, Georgia, serif", "font-size": "26px"}
	if seat.Ready {
		style["border"], style["border-radius"] = "1px solid #e7c27a", "9px"
		style["background"] = "linear-gradient(180deg,#d9a441,#8c541d)"
		style["box-shadow"] = "0 0 20px rgba(217,164,65,.3)"
		style["color"] = "#211a12"
	} else {
		label = "Awaiting Roll"
		if seat.HasStats {
			label = "Confirm on your phone"
		}
		style["border"], style["border-radius"] = "1px solid rgba(184,159,120,.4)", "9px"
		style["background"] = "rgba(16,22,30,.78)"
		style["box-shadow"] = "inset 0 0 0 1px rgba(217,164,65,.12)"
		style["color"] = "#948c7a"
	}
	return html.Div(html.Props{Class: "df-dm-creation-lockup", Role: "status", Aria: map[string]string{"live": "polite"}, Style: style}, html.Text("✦  "+label))
}
