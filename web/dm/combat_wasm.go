//go:build js && wasm

package dm

import (
	"strconv"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CombatComponent renders the FLAT battlefield with its projected grid and tokens.
func CombatComponent(view *dungeonfluxv1.DMView) router.Component {
	model := CombatModelFromView(view)
	locale := localeOrDefault(view.GetLocale())
	return func(_ router.Attrs) *router.Element {
		children := []ui.Node{combatGrid(model.Segments)}
		children = append(children, combatTokens(model.Tokens)...)
		return html.Section(html.Props{Class: "df-dm-combat", Role: "img", Aria: map[string]string{"label": T(locale, "dm.combat_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-combat-stage", Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "background-image": "url('" + model.ImageURL + "')", "background-size": "cover", "background-position": "center"}}, children...),
		)
	}
}

func combatGrid(segments []CombatSegment) ui.Node {
	children := make([]ui.Node, 0, len(segments))
	for _, segment := range segments {
		children = append(children, html.Line(html.Props{Raw: map[string]any{"x1": number(segment.From.X), "y1": number(segment.From.Y), "x2": number(segment.To.X), "y2": number(segment.To.Y)}}))
	}
	return html.Svg(html.Props{Class: "df-dm-combat-grid", Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "pointer-events": "none"}, Raw: map[string]any{"viewBox": "0 0 1920 1080", "preserveAspectRatio": "none", "aria-hidden": "true"}}, children...)
}

func combatTokens(tokens []CombatToken) []ui.Node {
	nodes := make([]ui.Node, 0, len(tokens))
	for _, token := range tokens {
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-token", Style: map[string]string{"position": "absolute", "left": percent(token.X / 1920 * 100), "top": percent(token.Y / 1080 * 100), "transform": "translate(-50%, -50%)", "width": "10%"}, Raw: map[string]any{"data-token-id": token.ID, "data-active": strconv.FormatBool(token.Active)}},
			html.Img(html.Props{Src: token.Portrait, Alt: token.Name, Class: "df-dm-combat-token-portrait", Style: map[string]string{"width": "100%", "height": "auto", "object-fit": "contain"}}),
			html.Div(html.Props{Class: "df-dm-combat-token-name"}, html.Text(token.Name)),
		))
	}
	return nodes
}
