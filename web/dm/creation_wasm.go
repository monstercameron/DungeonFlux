//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CreationComponent renders the 16:9 character-creation tableau for the DM.
// The TV is a read-only projection: phone choices are shown as live labels,
// while generated portraits and crests are resolved through ArtURL.
func CreationComponent(model CreationModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		featured := featuredCreationSeat(model)
		return html.Section(html.Props{Class: "df-dm-creation", Role: "region", Aria: map[string]string{"label": "Character creation"}, Style: creationStyle()},
			creationHeader(model),
			html.Div(html.Props{Class: "df-dm-creation-layout", Style: creationLayoutStyle()},
				creationPortraitPanel(featured),
				creationBuildPanel(featured),
				creationPhonePanel(featured),
			),
			creationSeatStrip(model.Seats, featured.Number),
			creationLockup(featured),
		)
	}
}

func creationStyle() map[string]string {
	background := ArtURL("ui/title_bg_wide")
	backgroundImage := "radial-gradient(ellipse at 50% 42%, rgba(7,10,16,.08), rgba(7,10,16,.74) 90%), linear-gradient(90deg, rgba(8,10,15,.78), rgba(8,10,15,.16) 55%, rgba(8,10,15,.82))"
	if background != "" {
		backgroundImage += ", url('" + background + "')"
	}
	return map[string]string{
		"position": "relative", "height": "100%", "min-height": "100%", "overflow": "hidden",
		"padding":    "clamp(1.2rem,3.2vw,3.6rem) clamp(1.5rem,4vw,5rem) clamp(1rem,2vw,2.2rem)",
		"background": backgroundImage, "background-position": "center", "background-size": "cover",
		"color": "#efe6d2", "font-family": "Inter,ui-sans-serif,system-ui,sans-serif",
	}
}

func creationHeader(model CreationModel) ui.Node {
	logo := ArtURL("ui/logo_wordmark")
	brand := ui.Node(html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": ".7rem", "min-height": "3.5rem"}},
		html.Span(html.Props{Style: map[string]string{"color": "#d9a441", "font-size": "clamp(1.4rem,2.2vw,2.5rem)"}}, html.Text("✦")),
		html.Span(html.Props{Style: map[string]string{"font-family": "Georgia,serif", "font-size": "clamp(1.25rem,2vw,2.3rem)", "font-weight": "700", "letter-spacing": ".04em"}}, html.Text("DungeonFlux")),
	))
	if logo != "" {
		brand = html.Img(html.Props{Src: logo, Alt: "DungeonFlux", Style: map[string]string{"display": "block", "width": "clamp(10rem,18vw,18rem)", "height": "auto", "object-fit": "contain"}})
	}
	return html.Header(html.Props{Class: "df-dm-creation-header", Style: map[string]string{"position": "relative", "z-index": "2", "display": "grid", "grid-template-columns": "1fr auto 1fr", "align-items": "start", "gap": "1rem", "text-align": "center"}},
		html.Div(html.Props{Style: map[string]string{"justify-self": "start"}}, brand),
		html.Div(html.Props{},
			html.P(html.Props{Style: map[string]string{"margin": ".15rem 0 .2rem", "color": "#d9a441", "font-size": "clamp(.62rem,1vw,1rem)", "font-weight": "700", "letter-spacing": ".28em", "text-transform": "uppercase"}}, html.Text("CHARACTER CREATION")),
			html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia,'Times New Roman',serif", "font-size": "clamp(2rem,4.2vw,5rem)", "font-weight": "600", "letter-spacing": ".02em", "line-height": "1"}}, html.Text("Create Your Character")),
			html.P(html.Props{Style: map[string]string{"margin": ".45rem 0 0", "color": "#efe6d2", "font-family": "Georgia,serif", "font-size": "clamp(.9rem,1.45vw,1.65rem)"}}, html.Text("Choose on your phone. The engine rolls the rest.")),
		),
		html.Div(html.Props{Style: map[string]string{"justify-self": "end", "max-width": "18rem", "padding-top": ".3rem", "color": "#a89f8c", "font-size": "clamp(.7rem,1vw,1rem)", "line-height": "1.25", "text-align": "right"}}, html.Text(model.Prompt)),
	)
}

func creationLayoutStyle() map[string]string {
	return map[string]string{"position": "relative", "z-index": "2", "display": "grid", "grid-template-columns": "minmax(0,1.1fr) minmax(0,.92fr) minmax(15rem,.72fr)", "gap": "clamp(.75rem,1.6vw,1.8rem)", "height": "clamp(25rem,64vh,42rem)", "margin-top": "clamp(.5rem,1.5vh,1.5rem)"}
}

func creationPanelStyle(extra map[string]string) map[string]string {
	background := "linear-gradient(145deg,rgba(12,16,23,.88),rgba(7,10,15,.94))"
	if frame := ArtURL("ui/panel_frame"); frame != "" {
		background += ", url('" + frame + "')"
	}
	style := map[string]string{
		"position": "relative", "min-width": "0", "overflow": "hidden", "border": "1px solid rgba(217,164,65,.72)",
		"border-radius": "10px", "background": background, "background-size": "cover",
		"box-shadow": "0 1rem 2.5rem rgba(0,0,0,.42), inset 0 0 2rem rgba(0,0,0,.32)",
	}
	for key, value := range extra {
		style[key] = value
	}
	return style
}

func creationPortraitPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle(map[string]string{"display": "flex", "flex-direction": "column", "justify-content": "flex-end", "padding": "clamp(.8rem,1.5vw,1.6rem)"})
	children := []ui.Node{creationCornerLabel("PLAYER " + strconv.Itoa(int(seat.Number)))}
	portrait := creationAssetURL(seat.PortraitURL)
	if portrait == "" {
		portrait = ArtURL(speciesArtName(seat.Species))
	}
	if portrait != "" {
		children = append(children, html.Img(html.Props{Src: portrait, Alt: creationDisplayValue(seat.Name, "Hero portrait"), Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "object-fit": "cover", "object-position": "center top", "opacity": ".9"}}))
	} else {
		children = append(children, html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "radial-gradient(circle at 50% 30%,rgba(48,73,102,.9),transparent 45%),linear-gradient(145deg,#111b2a,#0b0d13)"}}))
	}
	children = append(children,
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "inset": "0", "background": "linear-gradient(180deg,transparent 35%,rgba(5,7,11,.12) 50%,rgba(5,7,11,.95) 100%)"}}),
		html.Div(html.Props{Style: map[string]string{"position": "relative", "z-index": "2", "padding": "clamp(.7rem,1.2vw,1.2rem) clamp(.9rem,1.6vw,1.7rem)", "border": "1px solid rgba(217,164,65,.72)", "background": "rgba(8,11,16,.84)", "text-align": "center"}},
			html.H2(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia,serif", "font-size": "clamp(1.4rem,2.5vw,3rem)", "font-weight": "500"}}, html.Text(creationDisplayValue(seat.Name, "Waiting for hero"))),
			html.P(html.Props{Style: map[string]string{"margin": ".35rem 0 0", "color": "#d9a441", "font-size": "clamp(.72rem,1vw,1rem)", "font-weight": "700", "letter-spacing": ".16em", "text-transform": "uppercase"}}, html.Text(creationDisplayValue(seat.Class, "Class pending"))),
		),
	)
	return html.Div(html.Props{Class: "df-dm-creation-portrait", Role: "img", Aria: map[string]string{"label": "Generated hero portrait"}, Style: style}, children...)
}

func creationBuildPanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle(map[string]string{"display": "flex", "flex-direction": "column", "align-items": "center", "padding": "clamp(1rem,2vw,2rem) clamp(.8rem,1.8vw,1.8rem)", "text-align": "center"})
	crest := seat.ClassCrestURL
	if crest == "" {
		crest = ArtURL(classArtName(seat.Class))
	}
	crestNode := ui.Node(html.Div(html.Props{Style: map[string]string{"width": "clamp(3.8rem,7vw,7rem)", "height": "clamp(3.8rem,7vw,7rem)", "margin": ".15rem auto .4rem", "display": "grid", "place-items": "center", "border-radius": "50%", "background": "radial-gradient(circle,#27384d,#0c121c)", "color": "#d9a441", "font-size": "clamp(2rem,4vw,4rem)"}}, html.Text("✧")))
	if crest != "" {
		crestNode = html.Img(html.Props{Src: crest, Alt: creationDisplayValue(seat.Class, "Class") + " crest", Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "contain"}})
	}
	return html.Div(html.Props{Class: "df-dm-creation-build", Style: style},
		creationCornerLabel("ENGINE GENERATED BUILD"), crestNode,
		html.H2(html.Props{Style: map[string]string{"margin": ".2rem 0 0", "font-family": "Georgia,serif", "font-size": "clamp(1.35rem,2.1vw,2.5rem)", "font-weight": "500"}}, html.Text(creationDisplayValue(seat.Class, "Awaiting roll"))),
		html.P(html.Props{Style: map[string]string{"max-width": "22rem", "margin": ".35rem auto 1rem", "color": "#a89f8c", "font-family": "Georgia,serif", "font-size": "clamp(.85rem,1.15vw,1.2rem)", "line-height": "1.25"}}, html.Text("The engine rolls a legal hero from your phone choices.")),
		creationBuildValue("SPECIES", creationDisplayValue(seat.Species, "Not chosen")),
		creationBuildValue("GENDER", creationDisplayValue(seat.Gender, "Not chosen")),
		creationBuildValue("STATUS", creationDisplayValue(seat.Status, "Waiting for player")),
		html.Div(html.Props{Style: map[string]string{"width": "100%", "height": "1.1rem", "margin-top": "auto", "background-image": dividerBackground(), "background-position": "center", "background-size": "100% 100%", "opacity": ".78"}}),
	)
}

func creationBuildValue(label, value string) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "1rem", "width": "100%", "padding": ".52rem .65rem", "border-bottom": "1px solid rgba(217,164,65,.25)", "font-size": "clamp(.72rem,1vw,1rem)", "text-align": "left"}},
		html.Small(html.Props{Style: map[string]string{"color": "#d9a441", "font-weight": "700", "letter-spacing": ".12em"}}, html.Text(label)),
		html.Span(html.Props{Style: map[string]string{"color": "#efe6d2", "font-weight": "600", "text-align": "right", "text-transform": "capitalize"}}, html.Text(value)),
	)
}

func creationPhonePanel(seat CreationSeat) ui.Node {
	style := creationPanelStyle(map[string]string{"padding": "clamp(1rem,1.7vw,1.7rem) clamp(.75rem,1.4vw,1.4rem)", "background": "linear-gradient(145deg,rgba(10,22,31,.94),rgba(5,12,19,.96))"})
	return html.Div(html.Props{Class: "df-dm-creation-phone", Style: style},
		creationCornerLabel("PLAYER "+strconv.Itoa(int(seat.Number))+" · PHONE PICKER"),
		html.H2(html.Props{Style: map[string]string{"margin": ".7rem 0 .25rem", "font-family": "Georgia,serif", "font-size": "clamp(1.25rem,2vw,2.15rem)", "font-weight": "500", "line-height": "1.05"}}, html.Text("Your phone controls the hero")),
		html.P(html.Props{Style: map[string]string{"margin": "0 0 1rem", "color": "#a89f8c", "font-size": "clamp(.75rem,1vw,1rem)"}}, html.Text("Three choices. One roll. A story begins.")),
		creationChoiceRow("1", "SPECIES", creationDisplayValue(seat.Species, "Choose species"), speciesArtURL(seat.Species)),
		creationChoiceRow("2", "GENDER", creationDisplayValue(seat.Gender, "Choose gender"), ""),
		creationChoiceRow("3", "CLASS", creationDisplayValue(seat.Class, "Choose class"), classArtURL(seat.Class)),
		html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "padding-top": "1rem", "color": "#d9a441", "font-size": "clamp(.75rem,1vw,1rem)", "font-style": "italic", "text-align": "center"}}, html.Text("The engine rolls your stats.")),
	)
}

func creationChoiceRow(number, label, value, icon string) ui.Node {
	children := []ui.Node{html.Div(html.Props{Style: map[string]string{"display": "grid", "place-items": "center", "width": "2rem", "height": "2rem", "border": "1px solid rgba(217,164,65,.58)", "border-radius": "50%", "color": "#d9a441", "font-family": "Georgia,serif", "font-size": "1.1rem"}}, html.Text(number))}
	if icon != "" {
		children = append(children, html.Img(html.Props{Src: icon, Alt: "", Style: map[string]string{"width": "2.7rem", "height": "2.7rem", "object-fit": "cover", "border-radius": "50%", "border": "1px solid rgba(217,164,65,.4)"}}))
	}
	children = append(children, html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1"}}, html.Small(html.Props{Style: map[string]string{"display": "block", "color": "#a89f8c", "font-size": ".64em", "font-weight": "700", "letter-spacing": ".14em"}}, html.Text(label)), html.Span(html.Props{Style: map[string]string{"display": "block", "margin-top": ".12rem", "overflow": "hidden", "color": "#efe6d2", "font-size": "clamp(.85rem,1.15vw,1.2rem)", "font-weight": "600", "text-overflow": "ellipsis", "text-transform": "capitalize", "white-space": "nowrap"}}, html.Text(value))))
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": ".6rem", "min-height": "clamp(3.2rem,6vh,4.8rem)", "margin-top": ".5rem", "padding": ".45rem .55rem", "border": "1px solid rgba(217,164,65,.4)", "border-radius": "7px", "background": "rgba(19,30,40,.72)"}}, children...)
}

func creationSeatStrip(seats [2]CreationSeat, featured int32) ui.Node {
	items := make([]ui.Node, 0, len(seats))
	for _, seat := range seats {
		if seat.Number == featured {
			continue
		}
		portrait := creationAssetURL(seat.PortraitURL)
		if portrait == "" {
			portrait = speciesArtURL(seat.Species)
		}
		avatar := ui.Node(html.Div(html.Props{Style: map[string]string{"width": "3rem", "height": "3rem", "border-radius": "50%", "background": "radial-gradient(circle,#304866,#0c1018)", "border": "1px solid rgba(217,164,65,.58)"}}))
		if portrait != "" {
			avatar = html.Img(html.Props{Src: portrait, Alt: "", Style: map[string]string{"width": "3rem", "height": "3rem", "border-radius": "50%", "object-fit": "cover", "border": "1px solid rgba(217,164,65,.58)"}})
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": ".6rem", "padding": ".35rem .7rem", "border": "1px solid rgba(217,164,65,.46)", "border-radius": "7px", "background": "rgba(8,11,16,.86)"}}, avatar, html.Div(html.Props{}, html.Strong(html.Props{Style: map[string]string{"display": "block", "font-family": "Georgia,serif", "font-size": "1.05rem"}}, html.Text(creationDisplayValue(seat.Name, "Seat "+strconv.Itoa(int(seat.Number))))), html.Small(html.Props{Style: map[string]string{"color": "#a89f8c"}}, html.Text(creationDisplayValue(seat.Status, "Waiting for player"))))))
	}
	return html.Div(html.Props{Class: "df-dm-creation-seat-strip", Style: map[string]string{"position": "absolute", "z-index": "3", "left": "clamp(1.5rem,4vw,5rem)", "bottom": "clamp(1rem,2.4vh,2rem)", "display": "flex", "gap": ".6rem"}}, items...)
}

func creationLockup(seat CreationSeat) ui.Node {
	background := ArtURL("ui/button_primary")
	style := map[string]string{"position": "absolute", "z-index": "4", "right": "clamp(1.5rem,4vw,5rem)", "bottom": "clamp(1rem,2.4vh,2rem)", "display": "flex", "align-items": "center", "justify-content": "center", "min-width": "clamp(15rem,22vw,26rem)", "min-height": "clamp(3rem,6vh,4.8rem)", "padding": ".7rem 1.3rem", "border": "1px solid #e4b85d", "border-radius": "9px", "background": "linear-gradient(180deg,#d9a441,#8c541d)", "box-shadow": "0 .35rem 1.2rem rgba(0,0,0,.46), 0 0 1.2rem rgba(217,164,65,.25)", "color": "#fff3d7", "font-family": "Georgia,serif", "font-size": "clamp(1.2rem,2vw,2rem)", "text-shadow": "0 1px 2px #4e2d0c"}
	if background != "" {
		style["background-image"] = "linear-gradient(rgba(217,164,65,.18),rgba(92,48,12,.32)), url('" + background + "')"
		style["background-size"] = "cover"
	}
	label := "Lock In Character"
	if !seat.Ready {
		label = "Awaiting Roll"
	}
	return html.Div(html.Props{Class: "df-dm-creation-lockup", Role: "status", Aria: map[string]string{"live": "polite"}, Style: style}, html.Span(html.Props{Style: map[string]string{"margin-right": ".55rem", "font-size": ".8em"}}, html.Text("✧")), html.Text(label))
}

func creationCornerLabel(value string) ui.Node {
	return html.P(html.Props{Style: map[string]string{"align-self": "stretch", "margin": "0 0 .4rem", "color": "#d9a441", "font-size": "clamp(.58rem,.8vw,.82rem)", "font-weight": "700", "letter-spacing": ".18em", "text-align": "center"}}, html.Text(value))
}

func speciesArtName(species string) string {
	return "ui/species_" + strings.ToLower(strings.TrimSpace(species))
}

func classArtName(className string) string {
	return "ui/class_" + strings.ToLower(strings.TrimSpace(className))
}

func speciesArtURL(species string) string {
	if strings.TrimSpace(species) == "" {
		return ""
	}
	return ArtURL(speciesArtName(species))
}

func classArtURL(className string) string {
	if strings.TrimSpace(className) == "" {
		return ""
	}
	return ArtURL(classArtName(className))
}

func creationAssetURL(value string) string {
	if strings.HasPrefix(strings.TrimSpace(value), "/assets/preview/") {
		return ""
	}
	return strings.TrimSpace(value)
}

func dividerBackground() string {
	if divider := ArtURL("ui/divider"); divider != "" {
		return "url('" + divider + "')"
	}
	return "linear-gradient(90deg,transparent,#d9a441 20%,#d9a441 80%,transparent)"
}
