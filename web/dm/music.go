package dm

import (
	"math"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

const lobbyPhase = "lobby"

const (
	lobbyMusicAsset   = "THEME_MAIN"
	lobbyStingerAsset = "OPENING_SWELL"
)

// LobbyAudioCue names the accepted build-time assets used by the DM lobby.
type LobbyAudioCue struct {
	BedAsset     string
	StingerAsset string
	BedGain      float32
	FadeInMS     int
	FadeOutMS    int
	PlayStinger  bool
	LoopBed      bool
}

// LobbyAudioPlan returns the lobby bed and one-shot title cue. The caller
// supplies whether this client has already played the stinger, so reconnects
// can reattach to the bed without replaying the title cue.
func LobbyAudioPlan(phase string, stingerPlayed bool) LobbyAudioCue {
	if phase != lobbyPhase {
		return LobbyAudioCue{FadeOutMS: 300}
	}
	return LobbyAudioCue{BedAsset: lobbyMusicAsset, StingerAsset: lobbyStingerAsset, BedGain: 0.6, FadeInMS: 400, FadeOutMS: 300, PlayStinger: !stingerPlayed, LoopBed: true}
}

// MusicModel is the browser-owned projection of a DM music view.
type MusicModel struct {
	TrackID     string
	URL         string
	LoopStartMS int64
	LoopEndMS   int64
	BPM         int32
	Level       float32
	Duck        float32
	Cue         string
}

// Active reports whether the view contains a playable music track.
func (m MusicModel) Active() bool { return m.TrackID != "" && m.URL != "" }

// DisplayLevel clamps a mixer level for a stable indicator width.
func (m MusicModel) DisplayLevel() float32 {
	if m.Level < 0 {
		return 0
	}
	if m.Level > 1 {
		return 1
	}
	return m.Level
}

// MusicModelFromView projects the music portion of a DM view.
func MusicModelFromView(view *dungeonfluxv1.DMView) MusicModel {
	if view == nil || view.GetMusic() == nil {
		return MusicModel{}
	}
	music := view.GetMusic()
	model := MusicModel{
		TrackID:     music.GetTrackId(),
		URL:         music.GetUrl(),
		LoopStartMS: music.GetLoopStartMs(),
		LoopEndMS:   music.GetLoopEndMs(),
		BPM:         music.GetBpm(),
		Level:       music.GetLevel(),
		Duck:        music.GetDuck(),
		Cue:         music.GetCue(),
	}
	model.LoopStartMS = nonNegative(model.LoopStartMS)
	model.LoopEndMS = nonNegative(model.LoopEndMS)
	if model.LoopEndMS > 0 && model.LoopEndMS < model.LoopStartMS {
		model.LoopEndMS = model.LoopStartMS
	}
	return model
}

// MusicBarDurationMS returns the duration of one 4/4 bar for a valid tempo.
func MusicBarDurationMS(bpm int32) int64 {
	if bpm <= 0 {
		return 0
	}
	return int64(math.Round(240000 / float64(bpm)))
}

// MusicTransition describes a desired switch at a loop's next bar line.
type MusicTransition struct {
	// AtMS is the elapsed track time at which the switch should begin.
	AtMS int64
	// DelayMS is the time from the current view to AtMS.
	DelayMS int64
	// DurationMS is one outgoing-track bar, used for the equal-power fade.
	DurationMS int64
	// Crossfade reports whether the outgoing and incoming tracks overlap.
	Crossfade bool
}

// PlanMusicTransition returns the next-bar transition for a changed track.
// A cue is preserved as metadata, but does not bypass bar alignment.
func PlanMusicTransition(current MusicModel, next MusicModel, elapsedMS int64) MusicTransition {
	if next.TrackID == "" || next.TrackID == current.TrackID {
		return MusicTransition{}
	}
	if current.TrackID == "" {
		return MusicTransition{AtMS: elapsedMS}
	}
	bar := MusicBarDurationMS(current.BPM)
	if bar == 0 {
		bar = MusicBarDurationMS(next.BPM)
	}
	if bar == 0 {
		return MusicTransition{AtMS: elapsedMS, Crossfade: false}
	}
	loopStart := current.LoopStartMS
	if elapsedMS < loopStart {
		elapsedMS = loopStart
	}
	position := (elapsedMS - loopStart) % bar
	at := elapsedMS
	if position != 0 {
		at += bar - position
	}
	return MusicTransition{AtMS: at, DelayMS: at - elapsedMS, DurationMS: bar, Crossfade: true}
}
