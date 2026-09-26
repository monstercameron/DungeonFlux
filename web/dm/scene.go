package dm

import (
	"strconv"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// SceneLayer is one positioned still in the scene composition.
type SceneLayer struct {
	ID        string
	URL       string
	X         float32
	Y         float32
	Scale     float32
	Highlight bool
}

// SceneCharacter is the compact identity card used beside the scene.
type SceneCharacter struct {
	PlayerNumber int32
	Name         string
	Class        string
	PortraitURL  string
}

// SceneModel contains the DM scene data needed by the browser renderer.
type SceneModel struct {
	BackgroundURL string
	Layers        []SceneLayer
	Characters    []SceneCharacter
}

// SceneModelFromView projects a wire DM view into an owned scene model.
// Slices are copied so a later watch message cannot mutate a rendered scene.
func SceneModelFromView(view *dungeonfluxv1.DMView) SceneModel {
	if view == nil {
		return SceneModel{}
	}
	model := SceneModel{BackgroundURL: view.GetBackgroundUrl()}
	model.Layers = make([]SceneLayer, 0, len(view.GetLayers()))
	for _, layer := range view.GetLayers() {
		if layer == nil {
			continue
		}
		model.Layers = append(model.Layers, SceneLayer{
			ID:        layer.GetId(),
			URL:       layer.GetUrl(),
			X:         layer.GetX(),
			Y:         layer.GetY(),
			Scale:     layer.GetScale(),
			Highlight: layer.GetHighlight(),
		})
	}
	model.Characters = make([]SceneCharacter, 0, len(view.GetBuildCards()))
	for _, card := range view.GetBuildCards() {
		if card == nil {
			continue
		}
		model.Characters = append(model.Characters, SceneCharacter{
			PlayerNumber: card.GetPlayerNumber(),
			Name:         card.GetName(),
			Class:        card.GetClassName(),
			PortraitURL:  card.GetPortraitUrl(),
		})
	}
	return model
}

// SceneLayerStyle returns the CSS positioning for a scene layer.
func SceneLayerStyle(layer SceneLayer) map[string]string {
	style := map[string]string{
		"left":      percent(layer.X),
		"top":       percent(layer.Y),
		"transform": "translate(-50%, -50%)",
	}
	if layer.Scale > 0 {
		style["--df-layer-scale"] = number(layer.Scale)
		style["transform"] = "translate(-50%, -50%) scale(var(--df-layer-scale))"
	}
	return style
}

func percent(value float32) string {
	return number(value) + "%"
}

func number(value float32) string {
	return strconv.FormatFloat(float64(value), 'f', -1, 32)
}
