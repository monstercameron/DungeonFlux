package content

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// SoundCue maps a phase and event to a generated or build-time sound asset.
type SoundCue struct {
	ID       string
	Phase    vocab.StateID
	Event    string
	Kind     vocab.SoundKind
	Prompt   string
	Seconds  float64
	Loop     bool
	Fallback string
}

// SoundCatalogue contains the authored demo sound cues.
type SoundCatalogue struct {
	Cues []SoundCue
}

// DefaultSoundCatalogue returns the SFX, ambience, and music requests used by
// the demo phases and events.
func DefaultSoundCatalogue() SoundCatalogue {
	return SoundCatalogue{Cues: []SoundCue{
		{ID: "sfx_tavern_ambience", Phase: vocab.StateOpening, Event: "scene.active", Kind: vocab.SoundSFX, Prompt: "seamless rain-soaked tavern ambience loop, distant room murmur and hearth, no music", Seconds: 10, Loop: true, Fallback: "silence"},
		{ID: "sfx_dice_roll", Phase: vocab.StateCheck, Event: "check.rolling", Kind: vocab.SoundSFX, Prompt: "tight fantasy dice rolling across a wooden table, three quick impacts", Seconds: 2, Fallback: "silence"},
		{ID: "sfx_check_success", Phase: vocab.StateResolution, Event: "check.success", Kind: vocab.SoundSFX, Prompt: "brief bright magical success chime, warm and understated", Seconds: 2, Fallback: "silence"},
		{ID: "sfx_check_failure", Phase: vocab.StateResolution, Event: "check.failure", Kind: vocab.SoundSFX, Prompt: "brief muted ominous failure sting, low bell and soft scrape", Seconds: 2, Fallback: "silence"},
		{ID: "sfx_stranger_sting", Phase: vocab.StateHookEvent, Event: "stranger.arrives", Kind: vocab.SoundSFX, Prompt: "short ominous stranger arrival sting, bowed metal shimmer and a distant bell", Seconds: 4, Fallback: "silence"},
		{ID: "sfx_door_burst", Phase: vocab.StateCombat, Event: "combat.started", Kind: vocab.SoundSFX, Prompt: "a heavy tavern door bursts open with a wet wind gust and wood impact", Seconds: 3, Fallback: "silence"},
		{ID: "sfx_sword_slash", Phase: vocab.StateCombat, Event: "attack.made", Kind: vocab.SoundSFX, Prompt: "fast clean sword slash through wet air, fantasy combat", Seconds: 2, Fallback: "silence"},
		{ID: "sfx_blade_hit", Phase: vocab.StateCombat, Event: "damage.applied", Kind: vocab.SoundSFX, Prompt: "sharp blade hit on a drowned corpse, wet impact", Seconds: 2, Fallback: "silence"},
		{ID: "sfx_cliffhanger_hit", Phase: vocab.StateCliffhanger, Event: "line.done", Kind: vocab.SoundSFX, Prompt: "one massive tower bell strike in D with a long ringing decay and low impact", Seconds: 5, Fallback: "silence"},
		{ID: "THEME_MAIN", Phase: vocab.StateLobby, Event: "session.play", Kind: vocab.SoundMusic, Prompt: "dark fantasy chamber folk main theme for a rain-soaked smugglers' river town, instrumental only", Seconds: 96, Loop: true, Fallback: "silence"},
		{ID: "CREATION_BED_LOOP", Phase: vocab.StateCreation, Event: "creation.active", Kind: vocab.SoundMusic, Prompt: "sparse quiet dark fantasy chamber folk bed for character creation, instrumental only, soft harp and celesta, no lead melody", Seconds: 96, Loop: true, Fallback: "THEME_MAIN"},
		{ID: "TAVERN_WARM_LOOP", Phase: vocab.StateOpening, Event: "opening.narration", Kind: vocab.SoundMusic, Prompt: "warm sparse smugglers' tavern underscore under spoken dialogue, harp, muted hand drum, low whistle rests, instrumental only", Seconds: 96, Loop: true, Fallback: "THEME_MAIN"},
		{ID: "STING_COMBAT_START", Phase: vocab.StateCombat, Event: "combat.started", Kind: vocab.SoundMusic, Prompt: "sudden drowned corpse ambush hit with low strings and frame drum, one downbeat then held note, instrumental only", Seconds: 4, Fallback: "sfx_door_burst"},
		{ID: "COMBAT_SKIRMISH_LOOP", Phase: vocab.StateCombat, Event: "combat.active", Kind: vocab.SoundMusic, Prompt: "driving fantasy tavern skirmish loop, large frame drums, cello and bass ostinato, hurdy-gurdy drone, bold whistle fragments, instrumental only", Seconds: 96, Loop: true, Fallback: "THEME_MAIN"},
		{ID: "STING_VICTORY", Phase: vocab.StateCombat, Event: "combat.slain", Kind: vocab.SoundMusic, Prompt: "hard-won tavern brawl victory sting, rising whistle, bright harp, final bell in D, instrumental only", Seconds: 6, Fallback: "sfx_check_success"},
		{ID: "STING_BELL_TOLL", Phase: vocab.StateCombat, Event: "combat.fled", Kind: vocab.SoundMusic, Prompt: "one deep tower bell toll in D with a long decay over rain, instrumental only", Seconds: 4, Fallback: "sfx_cliffhanger_hit"},
	}}
}

// Find returns the first cue matching phase and event.
func (c SoundCatalogue) Find(phase vocab.StateID, event string) (SoundCue, bool) {
	for _, cue := range c.Cues {
		if cue.Phase == phase && cue.Event == event {
			return cue, true
		}
	}
	return SoundCue{}, false
}

// Validate checks cue IDs, prompts, duration bounds, and duplicate mappings.
func (c SoundCatalogue) Validate() error {
	if len(c.Cues) == 0 {
		return fmt.Errorf("sound catalogue is empty")
	}
	seenID := make(map[string]bool, len(c.Cues))
	seenEvent := make(map[string]bool, len(c.Cues))
	for _, cue := range c.Cues {
		if cue.ID == "" || cue.Event == "" || cue.Prompt == "" || cue.Seconds <= 0 || cue.Seconds > 600 || seenID[cue.ID] {
			return fmt.Errorf("invalid sound cue %q", cue.ID)
		}
		key := string(cue.Phase) + "\x00" + cue.Event + "\x00" + string(cue.Kind)
		if seenEvent[key] {
			return fmt.Errorf("duplicate sound cue mapping %q", key)
		}
		seenID[cue.ID], seenEvent[key] = true, true
	}
	return nil
}
