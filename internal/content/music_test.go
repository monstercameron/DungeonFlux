package content

import "testing"

func TestDefaultMusicCatalogue_ValidatesAndContainsDemoTracks(t *testing.T) {
	catalogue := DefaultMusicCatalogue()
	if err := catalogue.Validate(); err != nil {
		t.Fatalf("default catalogue should validate: %v", err)
	}
	if catalogue.Tracks[0].ID != "THEME_MAIN" || catalogue.Tracks[6].ID != "COMBAT_SKIRMISH_LOOP" || catalogue.Tracks[11].ID != "END_CARD_THEME" {
		t.Fatalf("unexpected generation order: %#v", catalogue.Tracks)
	}
	if got := len(catalogue.DomainTracks()); got != 12 {
		t.Fatalf("DomainTracks length = %d, want 12", got)
	}
}

func TestMusicCatalogue_LobbyTracksUseAcceptedAssets(t *testing.T) {
	bed, stinger, ok := DefaultMusicCatalogue().LobbyTracks()
	if !ok || bed.ID != "THEME_MAIN" || bed.Asset != "THEME_MAIN" || stinger.ID != "OPENING_SWELL" || stinger.Asset != "OPENING_SWELL" {
		t.Fatalf("lobby tracks = %#v, %#v, ok=%v", bed, stinger, ok)
	}
}

func TestMusicCatalogue_ValidateRejectsMalformedMetadata(t *testing.T) {
	tests := []struct {
		name string
		edit func(*MusicCatalogue)
	}{
		{"track count", func(c *MusicCatalogue) { c.Tracks = c.Tracks[:1] }},
		{"duplicate ID", func(c *MusicCatalogue) { c.Tracks[1].ID = c.Tracks[0].ID }},
		{"missing fallback", func(c *MusicCatalogue) { c.Tracks[0].Fallback = "" }},
		{"bad loop", func(c *MusicCatalogue) { c.Tracks[0].LoopEndMS = 1000 }},
		{"bad tempo", func(c *MusicCatalogue) { c.Tracks[6].BPM = 100 }},
		{"bad seed", func(c *MusicCatalogue) { c.Tracks[0].Seed++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			catalogue := DefaultMusicCatalogue()
			test.edit(&catalogue)
			if err := catalogue.Validate(); err == nil {
				t.Fatal("Validate accepted malformed catalogue")
			}
		})
	}
}
