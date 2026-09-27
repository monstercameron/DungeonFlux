package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestAudioEffects_targetsVoiceAndDM(t *testing.T) {
	tests := []struct {
		name  string
		event AudioEvent
		voice string
		dm    string
		seat  domain.SeatID
	}{
		{"attack", AudioEvent{Kind: vocab.EventAttackMade, Seat: 1, Class: "paladin", VoicePack: true}, "voicepack/1/attack_effort", "sfx_sword_slash", 1},
		{"damage", AudioEvent{Kind: vocab.EventDamageApplied, Target: 2, VoicePack: true}, "voicepack/2/hurt", "sfx_blade_hit", 2},
		{"spell", AudioEvent{Kind: "spell_cast", Seat: 2, VoicePack: true}, "voicepack/2/spell_cast", "sfx_spell_cast", 2},
		{"down", AudioEvent{Kind: vocab.EventStatusApplied, Target: 1, Status: vocab.StatusDown, VoicePack: true}, "voicepack/1/downed", "sfx_down", 1},
		{"victory", AudioEvent{Kind: vocab.EventCombatEnded, Seat: 2, Outcome: "SLAIN", VoicePack: true}, "voicepack/2/victory", "STING_VICTORY", 2},
		{"heal", AudioEvent{Kind: "heal", Target: 1, VoicePack: true}, "voicepack/1/heal", "sfx_heal", 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effects := AudioEffects(tt.event)
			if len(effects) != 2 {
				t.Fatalf("effects = %#v", effects)
			}
			voice := effects[0].(domain.PlaySound)
			if voice.Name != tt.voice || voice.Target != audioTargetSeat || voice.Seat != tt.seat {
				t.Fatalf("voice = %#v", voice)
			}
			dm := effects[1].(domain.PlaySound)
			if dm.Name != tt.dm || dm.Target != audioTargetDM || dm.Seat != 0 {
				t.Fatalf("dm = %#v", dm)
			}
		})
	}
}

func TestAudioEffects_usesClassFallbackAndRejectsNonCues(t *testing.T) {
	effects := AudioEffects(AudioEvent{Kind: "ability_used", Seat: 1, Class: "wizard"})
	if got := effects[0].(domain.PlaySound).Name; got != "sfx_wizard_spell_cast" {
		t.Fatalf("fallback = %q", got)
	}
	for _, event := range []AudioEvent{
		{Kind: vocab.EventStatusApplied, Target: 1, Status: vocab.StatusBloodied},
		{Kind: vocab.EventCombatEnded, Seat: 1, Outcome: "FLED"},
		{Kind: vocab.EventAttackMade, Seat: 3},
		{Kind: "unknown", Seat: 1},
	} {
		if got := AudioEffects(event); got != nil {
			t.Fatalf("event %#v produced %#v", event, got)
		}
	}
}
