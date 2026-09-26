//go:build js && wasm

package dm

import (
	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// SceneComponent renders a TV-safe 16:9 scene from a DM view snapshot.
func SceneComponent(view *dungeonfluxv1.DMView) router.Component {
	model := SceneModelFromView(view)
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{Class: "df-dm-scene", Role: "img", Aria: map[string]string{"label": "DungeonFlux scene"}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-scene-stage", Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "background-image": "url('" + model.BackgroundURL + "')", "background-size": "cover", "background-position": "center"}}, sceneLayers(model.Layers)...),
			html.Div(html.Props{Class: "df-dm-scene-cards", Role: "list", Aria: map[string]string{"label": "Heroes in the scene"}, Style: map[string]string{"position": "absolute", "inset": "0", "pointer-events": "none"}}, sceneCharacters(model.Characters)...),
		)
	}
}

func sceneLayers(layers []SceneLayer) []ui.Node {
	nodes := make([]ui.Node, 0, len(layers))
	for _, layer := range layers {
		class := "df-dm-scene-layer"
		if layer.Highlight {
			class += " is-highlighted"
		}
		style := SceneLayerStyle(layer)
		style["position"] = "absolute"
		nodes = append(nodes, html.Img(html.Props{ID: layer.ID, Class: class, Src: layer.URL, Alt: "", Style: style, Raw: map[string]any{"aria-hidden": "true"}}))
	}
	return nodes
}

func sceneCharacters(characters []SceneCharacter) []ui.Node {
	nodes := make([]ui.Node, 0, len(characters))
	for _, character := range characters {
		nodes = append(nodes, html.Div(html.Props{Class: "df-dm-scene-card", Role: "listitem", Style: map[string]string{"position": "relative", "display": "inline-flex", "flex-direction": "column"}},
			html.Img(html.Props{Src: character.PortraitURL, Alt: character.Name, Class: "df-dm-scene-card-portrait", Style: map[string]string{"max-width": "18vw", "max-height": "24vh", "object-fit": "contain"}}),
			html.Div(html.Props{Class: "df-dm-scene-card-copy"},
				html.Strong(html.Props{}, ui.Text(character.Name)),
				html.Small(html.Props{}, ui.Text(character.Class)),
			),
		))
	}
	return nodes
}
