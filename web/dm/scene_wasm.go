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
	locale := localeOrDefault(view.GetLocale())
	return func(_ router.Attrs) *router.Element {
		return html.Section(html.Props{Class: "df-dm-scene", Role: "img", Aria: map[string]string{"label": T(locale, "dm.scene_label", nil)}, Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "overflow": "hidden"}},
			html.Div(html.Props{Class: "df-dm-scene-stage", Style: map[string]string{"position": "relative", "width": "100%", "height": "100%", "background-image": "linear-gradient(180deg, rgba(15,17,23,.08) 35%, rgba(15,17,23,.82) 100%), url('" + model.BackgroundURL + "')", "background-size": "cover", "background-position": "center"}}, sceneLayers(model.Layers)...),
			html.Div(html.Props{Class: "df-dm-scene-cards", Role: "list", Aria: map[string]string{"label": T(locale, "dm.scene_heroes", nil)}, Style: map[string]string{"position": "absolute", "left": "3.5%", "bottom": "18%", "display": "flex", "align-items": "flex-end", "gap": "1.2rem", "pointer-events": "none"}}, sceneCharacters(model.Characters)...),
			sceneCaption(model.Caption),
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
		if layer.Speaking {
			class += " is-speaking"
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

func sceneCaption(caption SceneCaption) ui.Node {
	if !caption.Visible {
		return html.Div(html.Props{Class: "df-dm-scene-caption", Hidden: true})
	}
	style := map[string]string{
		"position": "absolute", "left": "3.5%", "right": "3.5%", "bottom": "3.5%",
		"padding": "1.1rem 1.5rem 1.25rem", "border": "1px solid rgba(217,164,65,.72)",
		"border-radius": "12px", "background": "linear-gradient(90deg, rgba(15,17,23,.96), rgba(26,29,38,.88))",
		"box-shadow": "0 10px 28px rgba(0,0,0,.38), inset 0 1px 0 rgba(239,230,210,.08)",
		"color":      "#efe6d2", "font-family": "Georgia, 'Times New Roman', serif", "z-index": "3",
	}
	nameStyle := map[string]string{"display": "block", "margin-bottom": ".28rem", "color": "#d9a441", "font-family": "Arial, sans-serif", "font-size": "clamp(.75rem, 1.1vw, 1.05rem)", "font-weight": "700", "letter-spacing": ".12em", "text-transform": "uppercase"}
	return html.Div(html.Props{Class: "df-dm-scene-caption", Role: "status", Aria: map[string]string{"live": "polite"}, Style: style},
		html.Strong(html.Props{Class: "df-dm-scene-caption-speaker", Style: nameStyle}, ui.Text(caption.Speaker)),
		html.P(html.Props{Class: "df-dm-scene-caption-text", Style: map[string]string{"margin": "0", "font-size": "clamp(1.35rem, 2.25vw, 2.65rem)", "line-height": "1.22", "font-weight": "600", "text-wrap": "balance"}}, ui.Text(caption.Text)),
	)
}
