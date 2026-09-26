package dm

import dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// ClipModel is the browser-independent playback choice for a DM clip.
type ClipModel struct {
	VideoURL    string
	StillURL    string
	OffsetMS    int64
	Playing     bool
	UseFallback bool
	Locale      string
}

// ClipStartSeconds returns the non-negative seek position for a browser video.
func (m ClipModel) ClipStartSeconds() float64 {
	return float64(m.OffsetMS) / 1000
}

// ClipModelFromView selects video when its asset is ready and a still otherwise.
func ClipModelFromView(view *dungeonfluxv1.DMView) ClipModel {
	if view == nil {
		return ClipModel{UseFallback: true}
	}
	model := ClipModel{StillURL: sceneBackgroundURL(view), UseFallback: true, Locale: view.GetLocale()}
	clip := view.GetClip()
	if clip == nil || clip.GetUrl() == "" {
		return model
	}
	model.VideoURL = clip.GetUrl()
	model.OffsetMS = nonNegative(clip.GetOffsetMs())
	model.Playing = clip.GetPlaying()
	model.UseFallback = clip.GetThen() == "STILL" || view.GetShot().GetFallback()
	return model
}

func nonNegative(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
