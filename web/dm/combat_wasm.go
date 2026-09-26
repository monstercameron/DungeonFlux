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
	return func(_ router.Attrs) *router.Element {
		children := []ui.Node{combatGrid(model.Segments)}
		children = append(children, combatTokens(model.Tokens)...)
		return html.Section(html.Props{Class: "df-dm-combat", Role: "img", Aria: map[string]string{"label": "Combat battlefield"}},
			html.Div(html.Props{Class: "df-dm-combat-stage", Style: map[string]string{"background-image": "url('" + model.ImageURL + "')"}}, children...),
		)
	}
}

func combatGrid(segments []CombatSegment) ui.Node {
	children := make([]ui.Node, 0, len(segments))
	for _, segment := range segments {
		children = append(children, html.Line(html.Props{Raw: map[string]any{"x1": number(segment.From.X), "y1": number(segment.From.Y), "x2": number(segment.To.X), "y2": number(segment.To.Y)}}))
	}
	return html.Svg(html.Props{Class: "df-dm-combat-grid", Raw: map[string]any{"viewBox": "0 0 1920 1080", "preserveAspectRatio": "none", "aria-hidden": "true"}}, children...)
}

func combatTokens(tokens []CombatToken) []ui.Node {
	nodes := make([]ui.Node, 0, len(tokens))
	for _, token := range tokens {
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-combat-token", Style: map[string]string{"left": number(token.X) + "px", "top": number(token.Y) + "px"}, Raw: map[string]any{"data-token-id": token.ID, "data-active": strconv.FormatBool(token.Active)}},
			html.Img(html.Props{Src: token.Portrait, Alt: token.Name, Class: "df-dm-combat-token-portrait"}),
			html.Div(html.Props{Class: "df-dm-combat-token-name"}, html.Text(token.Name)),
		))
	}
	return nodes
}
