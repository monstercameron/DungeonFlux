//go:build js && wasm

package dm

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/web/shell/audio"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const lobbyMusicFadeOutMS = 300

// MusicIndicator renders the operator-facing music state without exposing
// mixer controls on the presentation screen.
func MusicIndicator(model MusicModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		if !model.Active() {
			return html.Div(html.Props{Class: "df-dm-music df-dm-music-idle", Aria: map[string]string{"label": "Music idle"}, Style: map[string]string{"display": "none"}})
		}
		width := fmt.Sprintf("%.0f%%", model.DisplayLevel()*100)
		label := "Music"
		if model.Cue != "" {
			label += " · " + model.Cue
		}
		return html.Div(html.Props{Class: "df-dm-music", Role: "status", Aria: map[string]string{"label": label}, Style: map[string]string{"position": "absolute", "right": "1.25rem", "bottom": "1.25rem", "display": "inline-flex", "align-items": "center", "gap": "0.55rem", "padding": "0.45rem 0.7rem", "border": "1px solid rgba(217,164,65,.55)", "border-radius": "999px", "background": "rgba(15,17,23,.78)", "color": "#efe6d2", "font-size": "0.8rem", "letter-spacing": "0.08em", "text-transform": "uppercase"}},
			html.Span(html.Props{Class: "df-dm-music-glyph", Aria: map[string]string{"hidden": "true"}}, ui.Text("♫")),
			html.Span(html.Props{Class: "df-dm-music-label"}, ui.Text(label)),
			html.Span(html.Props{Class: "df-dm-music-meter", Style: map[string]string{"display": "inline-block", "width": "2.5rem", "height": "0.2rem", "border-radius": "999px", "background": "linear-gradient(90deg, #d9a441 " + width + ", rgba(239,230,210,.2) " + width + ")"}, Aria: map[string]string{"hidden": "true"}}),
		)
	}
}

// PlayLobbyAudio resolves accepted assets through the shell's gRPC-backed
// Blob URL source and schedules them on the Listen mixer. Sharing the mixer
// deduplicates streamed server cues and the autoplay fallback by asset name.
func PlayLobbyAudio(player *audio.Player, phase string) error {
	if player == nil {
		return fmt.Errorf("dm: audio player is required")
	}
	base := LobbyAudioPlan(lobbyPhase, false)
	plan := LobbyAudioPlan(phase, player.HasTrack(base.StingerAsset))
	if plan.BedAsset == "" {
		player.StopTrack(base.BedAsset, lobbyMusicFadeOutMS)
		return nil
	}
	if bedURL := ArtURL(plan.BedAsset); bedURL != "" {
		if err := player.PlayURL(plan.BedAsset, audio.MusicChannel, bedURL, plan.LoopBed, plan.BedGain, plan.FadeInMS); err != nil {
			return err
		}
	}
	if plan.PlayStinger {
		if stingerURL := ArtURL(plan.StingerAsset); stingerURL != "" {
			return player.PlayURL(plan.StingerAsset, audio.SFXChannel, stingerURL, false, 1, 0)
		}
	}
	return nil
}

// ResumeAudio unlocks the shared audio context after the table-audio button
// receives a user gesture.
func ResumeAudio(player *audio.Player) error {
	if player == nil {
		return fmt.Errorf("dm: audio player is required")
	}
	return player.Resume()
}
