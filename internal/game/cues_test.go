package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestDemoCues_coversEveryDemoState(t *testing.T) {
	want := []vocab.StateID{
		vocab.StateLobby, vocab.StateCreation, vocab.StateOpening,
		vocab.StateExploration, vocab.StateConversation, vocab.StateCheck,
		vocab.StateResolution, vocab.StateHookEvent, vocab.StateCombat,
		vocab.StateCliffhanger, vocab.StateEnd,
	}
	got := DemoCues()
	if len(got) != len(want) {
		t.Fatalf("cue count = %d, want %d", len(got), len(want))
	}
	for i, cue := range got {
		if cue.State != want[i] {
			t.Errorf("cue %d state = %q, want %q", i, cue.State, want[i])
		}
		if cue.MusicTrack == "" {
			t.Errorf("cue %q has no music track", cue.State)
		}
		if cue.Ambience == "" {
			t.Errorf("cue %q has no ambience", cue.State)
		}
	}
}

func TestCueEffect_EffectsTargetsDMAndPreservesAudioOrder(t *testing.T) {
	effects := CueForState(vocab.StateCombat).Effects()
	if len(effects) != 4 {
		t.Fatalf("combat effects = %d, want 4", len(effects))
	}
	want := []struct {
		channel vocab.SoundKind
		name    string
		target  string
		loop    bool
		gain    float32
		delay   int
	}{
		{vocab.SoundMusic, "STING_COMBAT_START", audioTargetDM, false, 0.8, 0},
		{vocab.SoundMusic, "COMBAT_SKIRMISH_LOOP", audioTargetDM, true, 0.5, 1500},
		{vocab.SoundAmbience, "ambience_wooded_path", audioTargetDM, true, 0.25, 0},
		{vocab.SoundSFX, "sfx_your_turn", audioTargetSeat, false, 0.9, 2500},
	}
	for index, expected := range want {
		got, ok := effects[index].(domain.PlaySound)
		if !ok || got.Channel != expected.channel || got.Name != expected.name || got.Target != expected.target || got.Loop != expected.loop || got.Gain != expected.gain || got.DelayMS != expected.delay {
			t.Fatalf("effect %d = %#v, want %#v", index, effects[index], expected)
		}
	}
}

func TestCueEffect_placesAndStingersFollowTheScript(t *testing.T) {
	tests := []struct {
		state    vocab.StateID
		ambience string
		stinger  string
	}{
		{vocab.StateLobby, "ambience_river_night", ""},
		{vocab.StateConversation, "ambience_tavern_rain", ""},
		{vocab.StateCheck, "ambience_tavern_rain", "sfx_dice_roll"},
		{vocab.StateHookEvent, "ambience_bell_tower_wind", "sfx_door_burst"},
		{vocab.StateCliffhanger, "ambience_bell_tower_wind", ""},
		{vocab.StateEnd, "ambience_dawn", ""},
	}
	for _, tc := range tests {
		t.Run(string(tc.state), func(t *testing.T) {
			cue := CueForState(tc.state)
			if cue.Ambience != tc.ambience || cue.Stinger != tc.stinger {
				t.Fatalf("cue = %#v", cue)
			}
		})
	}
	end := CueForState(vocab.StateEnd).Effects()
	hit, theme := end[0].(domain.PlaySound), end[1].(domain.PlaySound)
	if hit.Name != "STING_CLIFF_HIT" || hit.Gain != stingerGain || theme.Name != "END_CARD_THEME" || theme.Loop || theme.DelayMS != 2500 {
		t.Fatalf("end effects = %#v", end)
	}
	opening := CueForState(vocab.StateOpening).Effects()
	swell, tavern := opening[0].(domain.PlaySound), opening[1].(domain.PlaySound)
	if swell.Gain != openingGain || tavern.DelayMS != 6000 || !tavern.Loop || tavern.Gain != 0.3 {
		t.Fatalf("opening effects = %#v", opening)
	}
	if check := CueForState(vocab.StateCheck).Effects(); check[len(check)-1].(domain.PlaySound).Gain != tableSFXGain {
		t.Fatalf("check stinger = %#v", check)
	}
}

func TestCueEffect_EffectsOmitsMissingStinger(t *testing.T) {
	effects := CueForState(vocab.StateOpening).Effects()
	if len(effects) != 3 {
		t.Fatalf("opening effects = %d, want 3", len(effects))
	}
}

func TestCueForState_returnsAuthoredCue(t *testing.T) {
	tests := []struct {
		name  string
		state vocab.StateID
		music string
		shot  string
		bar   int
	}{
		{name: "opening", state: vocab.StateOpening, music: "OPENING_SWELL", shot: "EST_WIDE_PUSH", bar: 3000},
		{name: "combat", state: vocab.StateCombat, music: "STING_COMBAT_START", shot: "BB_LOOP", bar: 1500},
		{name: "cliffhanger", state: vocab.StateCliffhanger, music: "CLIFF_TENSION_BED", shot: "CLIFF_TWO_PUSH", bar: 4000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cue := CueForState(tt.state)
			if cue.MusicTrack != tt.music || cue.Shot != tt.shot || cue.BarMS != tt.bar {
				t.Fatalf("cue = %#v", cue)
			}
		})
	}
}

func TestCueEffect_StingersHandOffToLoopBeds(t *testing.T) {
	for _, tt := range []struct {
		name, state, oneShot, loop string
	}{
		{name: "opening", state: string(vocab.StateOpening), oneShot: "OPENING_SWELL", loop: "TAVERN_WARM_LOOP"},
		{name: "hook", state: string(vocab.StateHookEvent), oneShot: "STING_STRANGER", loop: "TAVERN_WARM_LOOP"},
		{name: "combat", state: string(vocab.StateCombat), oneShot: "STING_COMBAT_START", loop: "COMBAT_SKIRMISH_LOOP"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cue := CueForState(vocab.StateID(tt.state))
			if cue.MusicTrack != tt.oneShot || cue.MusicLoop != tt.loop {
				t.Fatalf("cue = %#v", cue)
			}
			effects := cue.Effects()
			first := effects[0].(domain.PlaySound)
			second := effects[1].(domain.PlaySound)
			if first.Name != tt.oneShot || first.Loop || second.Name != tt.loop || !second.Loop {
				t.Fatalf("music effects = %#v", effects[:2])
			}
		})
	}
}

func TestCueForState_unknownStateIsEmpty(t *testing.T) {
	if got := CueForState(vocab.StateID("not-a-state")); got != (CueEffect{}) {
		t.Fatalf("unknown cue = %#v", got)
	}
}

func TestCueForState_returnsIndependentCatalogueValue(t *testing.T) {
	first := CueForState(vocab.StateOpening)
	first.MusicTrack = "changed"
	second := CueForState(vocab.StateOpening)
	if second.MusicTrack != "OPENING_SWELL" {
		t.Fatalf("catalogue changed through returned value: %#v", second)
	}
}
