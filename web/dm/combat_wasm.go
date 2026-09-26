//go:build js && wasm

package dm

import (
	"strconv"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CombatComponent renders the FLAT battlefield with its projected grid and tokens.
func CombatComponent(view *dungeonfluxv1.DMView) router.Component {
	model := CombatModelFromView(view)
	locale := localeOrDefault(view.GetLocale())
	imageURL := combatImageURL(model)
	return func(_ router.Attrs) *router.Element {
		children := []ui.Node{combatGrid(model.Segments), combatHighlights(model.Highlights)}
		children = append(children, combatTokens(model.Tokens)...)
		children = append(children, combatHUD(model))
		style := map[string]string{"position": "relative", "width": "100%", "height": "100%", "background-size": "cover", "background-position": "center", "background-color": "#11141d", "background-image": "linear-gradient(180deg, rgba(15,17,23,.06), rgba(15,17,23,.3))"}
		if imageURL != "" {
			style["background-image"] = "linear-gradient(180deg, rgba(15,17,23,.06), rgba(15,17,23,.3)), url('" + imageURL + "')"
		}
		return html.Section(html.Props{Class: "df-dm-combat", Role: "img", Aria: map[string]string{"label": T(locale, "dm.combat_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-combat-stage", Style: style}, children...),
		)
	}
}

func combatImageURL(model CombatModel) string {
	if artURL := ArtURL("battlefield_tavern_flat"); artURL != "" {
		return artURL
	}
	return model.ImageURL
}

func combatGrid(segments []CombatSegment) ui.Node {
	children := make([]ui.Node, 0, len(segments))
	for _, segment := range segments {
		children = append(children, html.Line(html.Props{Raw: map[string]any{"x1": number(segment.From.X), "y1": number(segment.From.Y), "x2": number(segment.To.X), "y2": number(segment.To.Y), "stroke": "#d9a441", "stroke-opacity": "0.52", "stroke-width": "3"}}))
	}
	return html.Svg(html.Props{Class: "df-dm-combat-grid", Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "pointer-events": "none"}, Raw: map[string]any{"viewBox": "0 0 1920 1080", "preserveAspectRatio": "none", "aria-hidden": "true"}}, children...)
}

func combatHighlights(points []CombatPoint) ui.Node {
	nodes := make([]ui.Node, 0, len(points))
	for _, point := range points {
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-highlight", Style: map[string]string{"position": "absolute", "left": percent(point.X / 1920 * 100), "top": percent(point.Y / 1080 * 100), "transform": "translate(-50%, -50%)", "width": "4.5%", "aspect-ratio": "1", "border": "3px solid #3aa39a", "border-radius": "50%", "background": "rgba(58,163,154,.18)", "box-sizing": "border-box"}}))
	}
	return html.Div(html.Props{Class: "df-dm-combat-highlights", Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none"}}, nodes...)
}

func combatTokens(tokens []CombatToken) []ui.Node {
	nodes := make([]ui.Node, 0, len(tokens))
	for _, token := range tokens {
		status := strings.Join(token.Statuses, " · ")
		label := token.Name
		if status != "" {
			label += " · " + status
		}
		imageChildren := []ui.Node{
			html.Img(html.Props{Src: token.Portrait, Alt: token.Name, Class: "df-dm-combat-token-portrait", Style: map[string]string{"width": "100%", "height": "auto", "object-fit": "contain"}}),
			html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "8%", "right": "8%", "bottom": "4px", "height": "10px", "background": "#291b1b", "border": "2px solid #efe6d2", "border-radius": "8px", "overflow": "hidden"}}, html.Div(html.Props{Style: map[string]string{"width": combatHPPercent(token.HP, token.HPMax), "height": "100%", "background": combatHPColor(token.HP, token.HPMax)}})),
		}
		imageChildren = append(imageChildren, combatStatusIcons(token.Statuses)...)
		tokenChildren := []ui.Node{html.Div(html.Props{Style: map[string]string{"position": "relative", "filter": tokenFilter(token.Statuses)}}, imageChildren...)}
		if crest := classCrest(token.Class); crest != "" {
			tokenChildren = append(tokenChildren, html.Img(html.Props{Src: crest, Alt: token.Class + " crest", Style: map[string]string{"position": "absolute", "right": "-10%", "top": "-8%", "width": "30%", "aspect-ratio": "1", "object-fit": "contain"}}))
		}
		tokenChildren = append(tokenChildren, html.Div(html.Props{Class: "df-dm-combat-token-name", Style: map[string]string{"font": "600 20px system-ui, sans-serif", "color": "#efe6d2", "text-align": "center", "text-shadow": "0 2px 5px #000"}}, html.Text(label)))
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-token", Style: map[string]string{"position": "absolute", "left": percent(token.X / 1920 * 100), "top": percent(token.Y / 1080 * 100), "transform": "translate(-50%, -50%)", "width": "10%"}, Raw: map[string]any{"data-token-id": token.ID, "data-active": strconv.FormatBool(token.Active)}}, tokenChildren...))
	}
	return nodes
}

func classCrest(className string) string {
	className = strings.ToLower(strings.TrimSpace(className))
	if className == "" {
		return ""
	}
	return ArtURL("ui/class_" + strings.ReplaceAll(className, " ", "_"))
}

func combatStatusIcons(statuses []string) []ui.Node {
	nodes := make([]ui.Node, 0, len(statuses))
	for _, status := range statuses {
		name := strings.ToLower(strings.TrimSpace(status))
		if name == "" {
			continue
		}
		if artURL := ArtURL("ui/status_" + name); artURL != "" {
			nodes = append(nodes, html.Img(html.Props{Src: artURL, Alt: status, Style: map[string]string{"width": "28px", "height": "28px", "object-fit": "contain"}}))
		}
	}
	return nodes
}

func combatHUD(model CombatModel) ui.Node {
	return html.Div(html.Props{Class: "df-dm-combat-hud", Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none", "color": "#efe6d2", "font-family": "system-ui, sans-serif"}},
		html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "2rem", "top": "2rem", "max-width": "48%", "padding": "16px 22px", "background": "rgba(15,17,23,.88)", "border": "2px solid #d9a441", "border-radius": "12px", "box-shadow": "0 8px 24px rgba(0,0,0,.45)"}},
			html.Div(html.Props{Style: map[string]string{"font": "700 30px Georgia, serif", "letter-spacing": ".04em"}}, html.Text(combatBanner(model))),
			html.Div(html.Props{Style: map[string]string{"font-size": "20px", "color": "#a89f8c", "margin-top": "4px"}}, html.Text(roundLabel(model.Round))),
		),
		turnOrder(model.TurnOrder),
	)
}

func turnOrder(entries []CombatTurn) ui.Node {
	items := make([]ui.Node, 0, len(entries))
	for index, entry := range entries {
		background := "rgba(15,17,23,.84)"
		border := "1px solid rgba(239,230,210,.3)"
		if entry.Active {
			background, border = "rgba(217,164,65,.2)", "2px solid #d9a441"
		}
		if entry.Done {
			background = "rgba(15,17,23,.55)"
		}
		items = append(items, html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "10px", "padding": "9px 12px", "background": background, "border": border, "border-radius": "10px", "opacity": combatOpacity(entry.Done)}},
			html.Div(html.Props{Style: map[string]string{"font-weight": "700", "color": "#d9a441", "width": "24px", "text-align": "center"}}, html.Text(strconv.Itoa(index+1))),
			html.Img(html.Props{Src: entry.Portrait, Alt: "", Style: map[string]string{"width": "44px", "height": "44px", "object-fit": "contain"}}),
			html.Div(html.Props{Style: map[string]string{"font-size": "21px", "font-weight": "700", "white-space": "nowrap"}}, html.Text(entry.Name)),
		))
	}
	return html.Div(html.Props{Class: "df-dm-combat-turn-order", Aria: map[string]string{"label": "Turn order"}, Style: map[string]string{"position": "absolute", "right": "2rem", "top": "2rem", "display": "flex", "flex-direction": "column", "gap": "8px", "min-width": "230px"}}, items...)
}

func combatBanner(model CombatModel) string {
	if strings.TrimSpace(model.Banner) != "" {
		return model.Banner
	}
	return "Combat"
}

func roundLabel(round int32) string {
	if round <= 0 {
		return "Turn order"
	}
	return "Round " + strconv.Itoa(int(round))
}

func combatHPPercent(hp, max int32) string {
	if max <= 0 || hp <= 0 {
		return "0%"
	}
	if hp >= max {
		return "100%"
	}
	return strconv.Itoa(int(hp*100/max)) + "%"
}

func combatHPColor(hp, max int32) string {
	if max > 0 && hp*2 <= max {
		return "#b3372f"
	}
	return "#3aa39a"
}

func tokenFilter(statuses []string) string {
	for _, status := range statuses {
		if strings.EqualFold(status, "down") || strings.EqualFold(status, "defeated") {
			return "grayscale(1) opacity(.7)"
		}
	}
	return "drop-shadow(0 6px 8px rgba(0,0,0,.6))"
}

func combatOpacity(done bool) string {
	if done {
		return "0.55"
	}
	return "1"
}
