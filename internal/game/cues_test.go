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
	if len(effects) != 3 {
		t.Fatalf("combat effects = %d, want 3", len(effects))
	}
	want := []struct {
		channel vocab.SoundKind
		name    string
		loop    bool
		gain    float32
	}{
		{vocab.SoundMusic, "COMBAT_SKIRMISH_LOOP", true, 0.5},
		{vocab.SoundAmbience, "ambience_combat_tension", true, 0.25},
		{vocab.SoundSFX, "sfx_door_burst", false, 1},
	}
	for index, expected := range want {
		got, ok := effects[index].(domain.PlaySound)
		if !ok || got.Channel != expected.channel || got.Name != expected.name || got.Target != audioTargetDM || got.Loop != expected.loop || got.Gain != expected.gain {
			t.Fatalf("effect %d = %#v, want %#v", index, effects[index], expected)
		}
	}
}

func TestCueEffect_EffectsOmitsMissingStinger(t *testing.T) {
	effects := CueForState(vocab.StateOpening).Effects()
	if len(effects) != 2 {
		t.Fatalf("opening effects = %d, want 2", len(effects))
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
		{name: "combat", state: vocab.StateCombat, music: "COMBAT_SKIRMISH_LOOP", shot: "BB_LOOP", bar: 1500},
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
