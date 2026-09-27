package dm

import (
	"fmt"
	"strings"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

// KillcamModel describes a single server-owned cinematic, independent of DOM.
type KillcamModel struct {
	Key, URL, Outcome, Attacker, Victim, Locale string
	OffsetMS, DurationMS                        int64
	Playing                                     bool
}

// KillcamFromView rejects unusable clips and expired reconnect snapshots.
func KillcamFromView(view *df.DMView) KillcamModel {
	clip := view.GetKillCam()
	if clip == nil || !killcamURL(clip.GetUrl()) || clip.GetDurationMs() != 4000 || clip.GetOffsetMs() >= clip.GetDurationMs() {
		return KillcamModel{}
	}
	if clip.GetOutcome() != "victory" && clip.GetOutcome() != "defeat" {
		return KillcamModel{}
	}
	return KillcamModel{Key: fmt.Sprintf("%d:%s", clip.GetSequence(), clip.GetUrl()), URL: clip.GetUrl(), Outcome: clip.GetOutcome(), Attacker: clip.GetAttacker(), Victim: clip.GetVictim(), OffsetMS: nonNegative(clip.GetOffsetMs()), DurationMS: clip.GetDurationMs(), Playing: clip.GetPlaying(), Locale: view.GetLocale()}
}

func killcamURL(url string) bool {
	return strings.HasPrefix(url, "/assets/") && strings.HasSuffix(url, ".mp4") && !strings.ContainsAny(strings.TrimPrefix(url, "/assets/"), "/\\?#")
}

// KillcamPreloads returns a bounded, de-duplicated list of local video URLs.
func KillcamPreloads(view *df.DMView) []string {
	var urls []string
	seen := map[string]bool{}
	for _, url := range view.GetPreload() {
		if killcamURL(url) && !seen[url] && len(urls) < 8 {
			urls = append(urls, url)
			seen[url] = true
		}
	}
	return urls
}
