package dm

import (
	"strconv"
	"strings"

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
	Speaking  bool
}

// SceneCharacter is the compact identity card used beside the scene.
type SceneCharacter struct {
	PlayerNumber int32
	Name         string
	Class        string
	PortraitURL  string
}

// SceneCaption is the lower-third line currently being spoken on the DM TV.
type SceneCaption struct {
	Speaker      string
	Text         string
	PlayerNumber int32
	Visible      bool
	Speaking     bool
}

// SceneModel contains the DM scene data needed by the browser renderer.
type SceneModel struct {
	BackgroundURL string
	Layers        []SceneLayer
	Characters    []SceneCharacter
	Caption       SceneCaption
}

// SceneModelFromView projects a wire DM view into an owned scene model.
// Slices are copied so a later watch message cannot mutate a rendered scene.
func SceneModelFromView(view *dungeonfluxv1.DMView) SceneModel {
	if view == nil {
		return SceneModel{}
	}
	model := SceneModel{BackgroundURL: view.GetBackgroundUrl(), Caption: SceneCaptionFromView(view)}
	speaker := normalizeCaptionText(model.Caption.Speaker)
	model.Layers = make([]SceneLayer, 0, len(view.GetLayers()))
	for _, layer := range view.GetLayers() {
		if layer == nil {
			continue
		}
		layerID := normalizeCaptionText(layer.GetId())
		model.Layers = append(model.Layers, SceneLayer{
			ID:        layer.GetId(),
			URL:       layer.GetUrl(),
			X:         layer.GetX(),
			Y:         layer.GetY(),
			Scale:     layer.GetScale(),
			Highlight: layer.GetHighlight(),
			Speaking:  speaker != "" && (layerID == speaker || containsCaptionWord(layerID, speaker)),
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

// SceneCaptionFromView chooses the streamed narration text, falling back to
// the stable subtitle text while a wire update is between narration chunks.
func SceneCaptionFromView(view *dungeonfluxv1.DMView) SceneCaption {
	if view == nil {
		return SceneCaption{}
	}
	narration := view.GetNarration()
	subtitle := view.GetSubtitle()
	caption := SceneCaption{PlayerNumber: subtitle.GetPlayerNumber()}
	if narration != nil {
		caption.Speaker = narration.GetSpeaker()
		caption.Text = narration.GetTextSoFar()
	}
	if caption.Text == "" && subtitle != nil {
		caption.Text = subtitle.GetText()
	}
	caption.Visible = normalizeCaptionText(caption.Text) != ""
	caption.Speaking = caption.Visible && normalizeCaptionText(caption.Speaker) != ""
	return caption
}

func normalizeCaptionText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func containsCaptionWord(value, want string) bool {
	return strings.Contains(value, want) || strings.Contains(want, value)
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
