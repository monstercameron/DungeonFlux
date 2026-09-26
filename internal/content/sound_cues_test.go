package content

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDefaultSoundCatalogue_ValidatesAndFindsPhaseEvents(t *testing.T) {
	catalogue := DefaultSoundCatalogue()
	if err := catalogue.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		phase  vocab.StateID
		event  string
		target string
	}{
		{vocab.StateCheck, "check.rolling", ""},
		{vocab.StateCombat, "combat.started", ""},
		{vocab.StateOpening, "scene.active", ""},
		{vocab.StateLobby, "player.joined.tv", "dm"},
		{vocab.StateCreation, "choice.tapped", "phone"},
		{vocab.StateCreation, "hero.roll", "dm"},
	} {
		if cue, ok := catalogue.Find(test.phase, test.event); !ok || cue.Prompt == "" || cue.Target != test.target {
			t.Fatalf("missing cue for %s/%s", test.phase, test.event)
		}
	}
	if _, ok := catalogue.Find(vocab.StateEnd, "missing"); ok {
		t.Fatal("found unknown cue")
	}
}

func TestSoundCatalogue_RejectsMalformedCue(t *testing.T) {
	for name, change := range map[string]func(*SoundCatalogue){
		"empty":        func(c *SoundCatalogue) { c.Cues = nil },
		"duplicate id": func(c *SoundCatalogue) { c.Cues[1].ID = c.Cues[0].ID },
		"duplicate event": func(c *SoundCatalogue) {
			c.Cues[1].Phase, c.Cues[1].Event, c.Cues[1].Target = c.Cues[0].Phase, c.Cues[0].Event, c.Cues[0].Target
		},
		"missing prompt": func(c *SoundCatalogue) { c.Cues[0].Prompt = "" },
		"bad duration":   func(c *SoundCatalogue) { c.Cues[0].Seconds = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			catalogue := DefaultSoundCatalogue()
			change(&catalogue)
			if err := catalogue.Validate(); err == nil {
				t.Fatal("Validate accepted malformed catalogue")
			}
		})
	}
}
