package game

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestUIAudioEffects_mapsAcceptedTableMoments(t *testing.T) {
	tests := []struct {
		name  string
		event domain.Event
		want  []string
	}{
		{name: "join", event: domain.Join{Seat: 1, JoinKind: "phone"}, want: []string{"sfx_join_tv"}},
		{name: "start", event: domain.HostCmd{Cmd: vocab.HostStart}, want: []string{"sfx_host_start"}},
		{name: "roll", event: domain.Act{Seat: 2, Move: vocab.MoveRollHero}, want: []string{"sfx_roll_reveal"}},
		{name: "lock", event: domain.PCLocked{Seat: 1}, want: []string{"sfx_hero_lock"}},
		{name: "non-phone-join", event: domain.Join{Seat: 1, JoinKind: "dm"}},
		{name: "choice", event: domain.Act{Seat: 1, Move: vocab.MoveSpecies}},
		{name: "portrait ready", event: domain.AssetReady{Slot: "portrait:2"}, want: []string{"sfx_portrait_ready"}},
		{name: "other asset", event: domain.AssetReady{Slot: "cliffhanger"}},
		{name: "talk start", event: domain.TalkStart{Seat: 1}, want: []string{"sfx_talk_open"}},
		{name: "talk without seat", event: domain.TalkStart{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effects := UIAudioEffects(tt.event)
			if len(effects) != len(tt.want) {
				t.Fatalf("effects = %#v, want %v", effects, tt.want)
			}
			for index, effect := range effects {
				cue, ok := effect.(domain.PlaySound)
				if !ok || cue.Channel != vocab.SoundSFX || cue.Target != uiAudioTargetDM || cue.Name != tt.want[index] {
					t.Fatalf("effect[%d] = %#v", index, effect)
				}
			}
		})
	}
}

func TestStateStep_UIAudioHooksOnlyAcceptedEvents(t *testing.T) {
	state := New(domain.OneShot{}, []byte("ui-audio"))
	joined := state.Step(domain.Envelope{Event: domain.Join{Seat: 1, JoinKind: "phone"}})
	if len(joined.Effects) != 1 || joined.Effects[0].(domain.PlaySound).Name != "sfx_join_tv" {
		t.Fatalf("join effects = %#v", joined.Effects)
	}
	rejected := state.Step(domain.Envelope{Event: domain.Act{Seat: 1, Move: vocab.MoveRollHero}})
	if rejected.Ack == nil || rejected.Ack.Accepted || len(rejected.Effects) != 0 {
		t.Fatalf("rejected roll = %#v", rejected)
	}
}
