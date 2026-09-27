package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// CueEffect is the audiovisual instruction emitted when the engine enters a
// demo state. BarMS tells a client the grid on which a loop transition lands.
// The game package keeps this value pure; the runtime adapts it to its view.
type CueEffect struct {
	State      vocab.StateID `json:"state"`
	MusicTrack string        `json:"music_track"`
	MusicLoop  string        `json:"music_loop,omitempty"`
	Ambience   string        `json:"ambience"`
	Stinger    string        `json:"stinger,omitempty"`
	Shot       string        `json:"shot"`
	BarMS      int           `json:"bar_ms"`
	Transition string        `json:"transition"`
	// LoopDelayMS holds MusicLoop back so the one-shot MusicTrack can speak
	// first: the opening swell crossfades into the tavern loop at 6 s.
	LoopDelayMS int `json:"loop_delay_ms,omitempty"`
}

// Effects converts a state cue into the sound effects consumed by the audio
// runtime. Every effect targets the DM table stream; transition timing stays
// on CueEffect for clients that schedule crossfades at the authored bar.
func (c CueEffect) Effects() []domain.Effect {
	return soundEffects(c)
}

// CueForState returns the authored music and shot cue for a demo state.
// Unknown states return an empty cue, which lets callers reject or log an
// invalid transition without inventing media.
func CueForState(state vocab.StateID) CueEffect {
	for _, cue := range DemoCues() {
		if cue.State == state {
			return cue
		}
	}
	return CueEffect{}
}

// DemoCues returns one cue for every state in the binding demo machine.
// Places set the ambience: the river town at night outside (lobby,
// creation), the rain-soaked tavern inside (opening through resolution),
// wind through the opened door as the stranger arrives, the Wooded Path for
// the outdoor fight, the bell tower for the cliffhanger, and dawn for the
// end card. Stingers are music one-shots; Stinger is an SFX on the table.
func DemoCues() []CueEffect {
	return []CueEffect{
		{State: vocab.StateLobby, MusicTrack: "THEME_MAIN", Ambience: "ambience_river_night", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateCreation, MusicTrack: "CREATION_BED_LOOP", Ambience: "ambience_river_night", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateOpening, MusicTrack: "OPENING_SWELL", MusicLoop: "TAVERN_WARM_LOOP", LoopDelayMS: 6000, Ambience: "ambience_tavern_rain", Shot: "EST_WIDE_PUSH", BarMS: 3000, Transition: "crossfade_at_6000ms"},
		{State: vocab.StateExploration, MusicTrack: "TAVERN_WARM_LOOP", Ambience: "ambience_tavern_rain", Shot: "NPC_MCU_STATIC", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateConversation, MusicTrack: "TAVERN_WARM_LOOP", Ambience: "ambience_tavern_rain", Shot: "NPC_MCU_STATIC", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateCheck, MusicTrack: "TAVERN_WARM_LOOP", Ambience: "ambience_tavern_rain", Stinger: "sfx_dice_roll", Shot: "CHECK_TENSION", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateResolution, MusicTrack: "TAVERN_WARM_LOOP", Ambience: "ambience_tavern_rain", Shot: "HERO_LOW_PUSH", BarMS: 3000, Transition: "bar"},
		{State: vocab.StateHookEvent, MusicTrack: "STING_STRANGER", MusicLoop: "TAVERN_WARM_LOOP", Ambience: "ambience_bell_tower_wind", Stinger: "sfx_door_burst", Shot: "ARRIVAL_DOOR_STATIC", Transition: "after_line"},
		{State: vocab.StateCombat, MusicTrack: "STING_COMBAT_START", MusicLoop: "COMBAT_SKIRMISH_LOOP", LoopDelayMS: 1500, Ambience: "ambience_wooded_path", Shot: "BB_LOOP", BarMS: 1500, Transition: "after_line"},
		{State: vocab.StateCliffhanger, MusicTrack: "CLIFF_TENSION_BED", Ambience: "ambience_bell_tower_wind", Shot: "CLIFF_TWO_PUSH", BarMS: 4000, Transition: "fade_300ms"},
		{State: vocab.StateEnd, MusicTrack: "STING_CLIFF_HIT", MusicLoop: "END_CARD_THEME", LoopDelayMS: 2500, Ambience: "ambience_dawn", BarMS: 3000, Transition: "fade_4000ms"},
	}
}
