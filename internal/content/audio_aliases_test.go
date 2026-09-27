package content

import "testing"

func TestCanonicalAudioName_ResolvesLegacyNames(t *testing.T) {
	tests := map[string]string{
		"music_theme_drowned_lantern": "THEME_MAIN",
		"music_combat_thrall":         "COMBAT_SKIRMISH_LOOP",
		"canned_cliffhanger_npc":      "canned_cliffhanger_vell",
		"canned_slain_by_seat1":       "canned_combat_slain_seat1",
		"canned_slain_by_seat2":       "canned_combat_slain_seat2",
		"already-canonical":           "already-canonical",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			if got := CanonicalAudioName(name); got != want {
				t.Fatalf("CanonicalAudioName(%q) = %q, want %q", name, got, want)
			}
		})
	}
}

func TestDefaultOneShot_UsesGeneratedAudioNames(t *testing.T) {
	story := DefaultOneShot()
	for _, track := range story.Music.Tracks {
		if got := CanonicalAudioName(string(track.Asset)); got != string(track.Asset) {
			t.Fatalf("track %q retains legacy asset %q", track.ID, track.Asset)
		}
	}
	for _, line := range CannedLines() {
		if got := CanonicalAudioName(line.ID); got != line.ID {
			t.Fatalf("canned line retains legacy asset %q", line.ID)
		}
	}
}
