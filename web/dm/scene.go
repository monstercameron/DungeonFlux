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
	LineID       string
	PlayerNumber int32
	Visible      bool
	Speaking     bool
	Done         bool
}

// SceneModel contains the DM scene data needed by the browser renderer.
type SceneModel struct {
	BackgroundURL      string
	FrameURL           string
	DividerURL         string
	SpeakerPortraitURL string
	Opening            bool
	Title              string
	Act                string
	Tagline            string
	Progress           []SceneProgress
	ProgressIndex      int
	ShowTitle          bool
	Layers             []SceneLayer
	Characters         []SceneCharacter
	Caption            SceneCaption
}

// SceneProgress is one story beat shown in the opening-scene rail.
type SceneProgress struct {
	Label     string
	Completed bool
	Active    bool
}

// SceneModelFromView projects a wire DM view into an owned scene model.
// Slices are copied so a later watch message cannot mutate a rendered scene.
func SceneModelFromView(view *dungeonfluxv1.DMView) SceneModel {
	if view == nil {
		return SceneModel{}
	}
	caption := SceneCaptionFromView(view)
	model := SceneModel{
		BackgroundURL:      sceneBackgroundURL(view),
		FrameURL:           artSrc("ui/panel_frame"),
		DividerURL:         artSrc("ui/divider"),
		SpeakerPortraitURL: speakerPortraitURL(caption.Speaker),
		Opening:            isOpeningView(view),
		Title:              SceneTitle(""),
		Act:                SceneAct(""),
		Tagline:            SceneTagline(""),
		Caption:            caption,
		ShowTitle:          !caption.Visible || isDMSpeaker(caption.Speaker),
	}
	if model.Opening && !caption.Visible {
		model.Caption = SceneCaption{Speaker: "Dungeon Master", Text: openingNarration, Visible: true, Speaking: true, Done: true}
		model.SpeakerPortraitURL = speakerPortraitURL(model.Caption.Speaker)
		model.ShowTitle = true
	}
	model.Progress, model.ProgressIndex = sceneProgress(view)
	speaker := normalizeCaptionText(model.Caption.Speaker)
	model.Layers = make([]SceneLayer, 0, len(view.GetLayers()))
	for _, layer := range view.GetLayers() {
		if layer == nil {
			continue
		}
		layerID := normalizeCaptionText(layer.GetId())
		model.Layers = append(model.Layers, SceneLayer{
			ID:        layer.GetId(),
			URL:       layerURL(layer.GetId(), layer.GetUrl()),
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
			PortraitURL:  artSrc(card.GetPortraitUrl()),
		})
	}
	return model
}

const openingNarration = "Rain drums against the warped roof of the Drowned Lantern, a tavern that smells of salt, smoke, and secrets. Lanterns sway above dark water pooling in the streets outside."

func isOpeningView(view *dungeonfluxv1.DMView) bool {
	if view == nil || view.GetClip() == nil {
		return false
	}
	value := normalizeCaptionText(view.GetClip().GetUrl())
	return strings.Contains(value, "opening") || strings.Contains(value, "establishing")
}

func sceneBackgroundURL(view *dungeonfluxv1.DMView) string {
	for _, name := range []string{"tavern_interior", "establishing_tavern"} {
		if value := artSrc(name); value != "" {
			return value
		}
	}
	return artSrc(view.GetBackgroundUrl())
}

func layerURL(id, fallback string) string {
	if fallback != "" {
		return artSrc(fallback)
	}
	switch {
	case strings.Contains(normalizeCaptionText(id), "vell"):
		return artSrc("mother_vell")
	case strings.Contains(normalizeCaptionText(id), "stranger"), strings.Contains(normalizeCaptionText(id), "courier"):
		return artSrc("stranger")
	default:
		return ""
	}
}

func speakerPortraitURL(speaker string) string {
	normalized := normalizeCaptionText(speaker)
	switch {
	case isDMSpeaker(speaker):
		return artSrc("ui/dm_speaker")
	case strings.Contains(normalized, "vell"):
		return artSrc("mother_vell")
	case strings.Contains(normalized, "stranger"), strings.Contains(normalized, "courier"):
		return artSrc("stranger")
	default:
		return ""
	}
}

func isDMSpeaker(speaker string) bool {
	normalized := normalizeCaptionText(speaker)
	return normalized == "dm" || normalized == "dungeon master" || normalized == "narrator"
}

func sceneProgress(view *dungeonfluxv1.DMView) ([]SceneProgress, int) {
	labels := SceneStepLabels("")
	index := 0
	switch {
	case view.GetBattlefield() != nil:
		index = 3
	case view.GetDice() != nil:
		index = 2
	case view.GetNarration() != nil && view.GetNarration().GetSpeaker() != "":
		index = 1
	case view.GetClip() != nil && strings.Contains(normalizeCaptionText(view.GetClip().GetUrl()), "stranger"):
		index = 1
	}
	progress := make([]SceneProgress, len(labels))
	for i, label := range labels {
		progress[i] = SceneProgress{Label: label, Completed: i < index, Active: i == index}
	}
	return progress, index
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
		caption.LineID = narration.GetLineId()
		caption.Done = narration.GetDone()
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
