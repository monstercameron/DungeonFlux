package content

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// Funnel conditions: the closed set of story events a plot thread can hang a
// rung on. They mirror the engine's steering outcomes without importing the
// engine, so content stays a pure data layer.
const (
	// CondEnter opens a thread when its beat is entered.
	CondEnter = "enter"
	// CondClueGranted opens gated material once the clue is earned by any
	// route: persuasion success or the stranger's relocated line.
	CondClueGranted = "clue_granted"
	// CondPersuadeSuccess fires when Mother Vell yields the clue.
	CondPersuadeSuccess = "persuade_success"
	// CondPersuadeFail fires when she refuses and the clue relocates.
	CondPersuadeFail = "persuade_fail"
	// CondCombatSlain fires when the thrall drops.
	CondCombatSlain = "combat_slain"
	// CondCombatFled fires when the bell tolls and the thrall flees.
	CondCombatFled = "combat_fled"
)

// KnownCondition reports whether a condition belongs to the closed set.
func KnownCondition(condition string) bool {
	switch condition {
	case CondEnter, CondClueGranted, CondPersuadeSuccess, CondPersuadeFail, CondCombatSlain, CondCombatFled:
		return true
	}
	return false
}

// FunnelRung is one step of a plot thread: the DM's live intro line for a
// story event, the prompt role that renders it, the clue it may carry, and
// the threads it leads to. IntroLine is the live-first opener; the canned
// fallbacks in canned.go cover the same moments when generation fails.
type FunnelRung struct {
	When       string   `json:"when"`
	IntroLine  string   `json:"intro_line"`
	WordCap    int      `json:"word_cap"`
	PromptRole string   `json:"prompt_role"`
	Clue       string   `json:"clue,omitempty"`
	Leads      []string `json:"leads,omitempty"`
}

// PlotThread funnels one story question down the demo path: each rung answers
// "what does the DM say when this happens" for exactly one outcome, so every
// branch (persuaded or refused, slain or fled) still arrives with the clue
// delivered and the next beat staged.
type PlotThread struct {
	ID       string       `json:"id"`
	BeatID   string       `json:"beat_id"`
	Synopsis string       `json:"synopsis"`
	Rungs    []FunnelRung `json:"rungs"`
}

// PlotThreads is the configurable thread set for one story.
type PlotThreads struct {
	Threads []PlotThread `json:"threads"`
}

// DefaultPlotThreads returns the Drowned Lantern funnel: five threads that
// carry the run from the opening narration to the midnight bell, with both
// persuasion outcomes and both combat outcomes scripted.
func DefaultPlotThreads() PlotThreads {
	return PlotThreads{Threads: []PlotThread{
		{
			ID:       "opening",
			BeatID:   "opening",
			Synopsis: "Set the rain, the missing lamplighter, and Mother Vell's watching eye.",
			Rungs: []FunnelRung{
				{When: CondEnter, IntroLine: "Rain hammers the Drowned Lantern, and the river keeps rising. The lamplighter never came home, and Mother Vell is watching the door with her one good eye.", WordCap: 40, PromptRole: "opening", Leads: []string{"find_lamplighter"}},
			},
		},
		{
			ID:       "find_lamplighter",
			BeatID:   "conversation",
			Synopsis: "Win the bell-tower clue from Mother Vell, or lose it to the stranger.",
			Rungs: []FunnelRung{
				{When: CondEnter, IntroLine: "Mother Vell polishes a glass that never gets clean. Ask her about the lamplighter, but she answers nothing for free.", WordCap: 25, PromptRole: "npc_reply", Leads: []string{"courier_hook"}},
				{When: CondPersuadeSuccess, IntroLine: "Mother Vell leans in despite herself. The lamplighter was dragged toward the old bell tower, and its bell rang at midnight.", WordCap: 25, PromptRole: "npc_reveal", Clue: "bell_tower_clue", Leads: []string{"courier_hook"}},
				{When: CondPersuadeFail, IntroLine: "Mother Vell turns back to her glasses. The direct road is closed; watch the door instead, because something wet just came in.", WordCap: 25, PromptRole: "npc_refuse", Leads: []string{"courier_hook"}},
			},
		},
		{
			ID:       "courier_hook",
			BeatID:   "stranger",
			Synopsis: "The courier's letter names a hero from their own hook, and carries the relocated clue when persuasion failed.",
			Rungs: []FunnelRung{
				{When: CondEnter, IntroLine: "The door bangs open on rain and river stink. A hooded courier stumbles in, clutching a sealed letter.", WordCap: 30, PromptRole: "stranger_lines"},
				{When: CondClueGranted, IntroLine: "The courier shakes river water off his hood and holds out a sealed letter. Whatever was in that water, it followed him here.", WordCap: 30, PromptRole: "stranger_lines", Leads: []string{"thrall_fight"}},
				{When: CondPersuadeFail, IntroLine: "The courier's letter speaks of the old bell tower, where something wet and dead stood guard. It followed him from the river.", WordCap: 30, PromptRole: "stranger_lines", Clue: "bell_tower_clue", Leads: []string{"thrall_fight"}},
			},
		},
		{
			ID:       "thrall_fight",
			BeatID:   "combat",
			Synopsis: "The drowned thrall bursts through the door; it drops or the bell sends it home.",
			Rungs: []FunnelRung{
				{When: CondEnter, IntroLine: "The tavern door bursts inward. A drowned thrall, bound to the tower bell, drags itself toward the light.", WordCap: 20, PromptRole: "combat_outcomes", Leads: []string{"midnight_bell"}},
				{When: CondCombatSlain, IntroLine: "Steel finds the thrall's heart of river mud, and it collapses into dark water.", WordCap: 20, PromptRole: "combat_outcomes", Leads: []string{"midnight_bell"}},
				{When: CondCombatFled, IntroLine: "The tower bell tolls once. The thrall turns mid-swing and lurches into the rain.", WordCap: 20, PromptRole: "combat_outcomes", Leads: []string{"midnight_bell"}},
			},
		},
		{
			ID:       "midnight_bell",
			BeatID:   "cliffhanger",
			Synopsis: "Midnight tolls, the lanterns die, and the ringer knows the heroes' names.",
			Rungs: []FunnelRung{
				{When: CondEnter, IntroLine: "The tower bell begins to toll midnight, slow and patient, and every lantern in the tavern gutters out.", WordCap: 40, PromptRole: "cliffhanger"},
				{When: CondClueGranted, IntroLine: "Midnight. The tower bell Mother Vell warned of tolls, and every lantern gutters out. Whoever pulls that rope already knows your names.", WordCap: 40, PromptRole: "cliffhanger"},
				{When: CondPersuadeFail, IntroLine: "Midnight. The courier's letter falls open as the tower bell tolls, and every lantern gutters out. Inside, in wet ink, are your names.", WordCap: 40, PromptRole: "cliffhanger"},
			},
		},
	}}
}

// LoadPlotThreads decodes a thread set from JSON data, so a new story ships
// as a data file rather than a code change.
func LoadPlotThreads(data []byte) (PlotThreads, error) {
	var threads PlotThreads
	if err := json.Unmarshal(data, &threads); err != nil {
		return PlotThreads{}, fmt.Errorf("decode plot threads: %w", err)
	}
	if err := threads.Validate(nil); err != nil {
		return PlotThreads{}, err
	}
	return threads, nil
}

// Validate checks the thread set against the story's beats: every thread
// binds a real beat, every rung names a known condition inside its word cap,
// every lead resolves to another thread, and the enter rung comes first so
// the funnel always has a staged opening.
func (t PlotThreads) Validate(beats []domain.Beat) error {
	if len(t.Threads) == 0 {
		return fmt.Errorf("plot needs at least one thread")
	}
	known := make(map[string]bool, len(t.Threads))
	for _, thread := range t.Threads {
		if thread.ID == "" {
			return fmt.Errorf("plot holds a thread without id")
		}
		if known[thread.ID] {
			return fmt.Errorf("duplicate plot thread %q", thread.ID)
		}
		known[thread.ID] = true
	}
	beatIDs := make(map[string]bool, len(beats))
	for _, beat := range beats {
		beatIDs[beat.ID] = true
	}
	for _, thread := range t.Threads {
		if thread.Synopsis == "" {
			return fmt.Errorf("thread %q needs a synopsis", thread.ID)
		}
		if len(beats) > 0 && !beatIDs[thread.BeatID] {
			return fmt.Errorf("thread %q binds unknown beat %q", thread.ID, thread.BeatID)
		}
		if len(thread.Rungs) == 0 {
			return fmt.Errorf("thread %q needs at least one rung", thread.ID)
		}
		if thread.Rungs[0].When != CondEnter {
			return fmt.Errorf("thread %q must stage its opening rung first", thread.ID)
		}
		seen := make(map[string]bool, len(thread.Rungs))
		for _, rung := range thread.Rungs {
			if !KnownCondition(rung.When) {
				return fmt.Errorf("thread %q hangs a rung on unknown condition %q", thread.ID, rung.When)
			}
			if seen[rung.When] {
				return fmt.Errorf("thread %q repeats condition %q", thread.ID, rung.When)
			}
			seen[rung.When] = true
			if rung.IntroLine == "" || rung.PromptRole == "" {
				return fmt.Errorf("thread %q has a rung without line or role", thread.ID)
			}
			if rung.WordCap <= 0 || len(strings.Fields(rung.IntroLine)) > rung.WordCap {
				return fmt.Errorf("thread %q breaks its word cap on %q", thread.ID, rung.When)
			}
			for _, lead := range rung.Leads {
				if !known[lead] {
					return fmt.Errorf("thread %q leads to unknown thread %q", thread.ID, lead)
				}
			}
		}
	}
	return nil
}

// RungFor returns the rung a thread stages for one story event.
func (t PlotThreads) RungFor(threadID, condition string) (FunnelRung, bool) {
	for _, thread := range t.Threads {
		if thread.ID != threadID {
			continue
		}
		for _, rung := range thread.Rungs {
			if rung.When == condition {
				return rung, true
			}
		}
	}
	return FunnelRung{}, false
}

// IntroFor is the DM's opener for one story event: the rung's intro line.
func (t PlotThreads) IntroFor(threadID, condition string) (string, bool) {
	rung, ok := t.RungFor(threadID, condition)
	if !ok {
		return "", false
	}
	return rung.IntroLine, true
}
