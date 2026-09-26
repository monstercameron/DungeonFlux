package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func TestAttackAudio_targetsAttackerAndDM(t *testing.T) {
	effects := AttackAudio(AttackResult{Seat: 2})
	if len(effects) != 2 {
		t.Fatalf("effects = %#v", effects)
	}
	voice := effects[0].(domain.PlaySound)
	if voice.Name != "voicepack/2/attack_effort" || voice.Target != "seat" || voice.Seat != 2 {
		t.Fatalf("voice = %#v", voice)
	}
	dm := effects[1].(domain.PlaySound)
	if dm.Name != "sfx_sword_slash" || dm.Target != "dm" {
		t.Fatalf("dm = %#v", dm)
	}
}

func TestEnemyAudio_hurtAndDownCues(t *testing.T) {
	result := EnemyResult{TargetSeat: 1, Outcome: rules.AttackOutcome{Hit: true, HPAfter: 0}}
	effects := EnemyAudio(result)
	if len(effects) != 4 {
		t.Fatalf("effects = %#v", effects)
	}
	if effects[0].(domain.PlaySound).Name != "voicepack/1/hurt" || effects[2].(domain.PlaySound).Name != "voicepack/1/downed" {
		t.Fatalf("target cues = %#v", effects)
	}
}

func TestEnemyAudio_missAndInvalidAreSilent(t *testing.T) {
	if got := EnemyAudio(EnemyResult{TargetSeat: 1}); got != nil {
		t.Fatalf("miss = %#v", got)
	}
	if got := EnemyAudio(EnemyResult{TargetSeat: 3, Outcome: rules.AttackOutcome{Hit: true, HPAfter: 1}}); got != nil {
		t.Fatalf("invalid = %#v", got)
	}
}

func TestVictoryAudio_targetsWinningSeat(t *testing.T) {
	if got := VictoryAudio(1)[0].(domain.PlaySound); got.Name != "voicepack/1/victory" || got.Seat != 1 {
		t.Fatalf("victory = %#v", got)
	}
	if VictoryAudio(0) != nil {
		t.Fatal("invalid victory should be silent")
	}
}
