package combat

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func sounds(t *testing.T, effects []domain.Effect) []domain.PlaySound {
	t.Helper()
	out := make([]domain.PlaySound, 0, len(effects))
	for _, effect := range effects {
		sound, ok := effect.(domain.PlaySound)
		if !ok {
			t.Fatalf("effect %#v is not a PlaySound", effect)
		}
		out = append(out, sound)
	}
	return out
}

func find(list []domain.PlaySound, name string) (domain.PlaySound, bool) {
	for _, sound := range list {
		if sound.Name == name {
			return sound, true
		}
	}
	return domain.PlaySound{}, false
}

func TestAttackAudio_schedulesSwingAndImpactOnTheTVTimeline(t *testing.T) {
	path := []Cell{{X: 1, Y: 1}, {X: 2, Y: 1}}
	tests := []struct {
		name    string
		outcome rules.AttackOutcome
		impact  string
		delay   int
		fall    bool
	}{
		{name: "even blade hit", outcome: rules.AttackOutcome{Hit: true, Natural: 14, HPAfter: 5, Damage: []rules.DamageRoll{{Type: "slashing"}}}, impact: "sfx_blade_hit", delay: 500 + attackContact},
		{name: "untyped blade hit", outcome: rules.AttackOutcome{Hit: true, Natural: 13, HPAfter: 5}, impact: "sfx_blade_hit", delay: 500 + attackContact},
		{name: "mace", outcome: rules.AttackOutcome{Hit: true, Natural: 12, HPAfter: 5, Damage: []rules.DamageRoll{{Type: "bludgeoning"}}}, impact: "sfx_mace_thud", delay: 500 + attackContact},
		{name: "dagger", outcome: rules.AttackOutcome{Hit: true, Natural: 12, HPAfter: 5, Damage: []rules.DamageRoll{{Type: "piercing"}}}, impact: "sfx_dagger_stab", delay: 500 + attackContact},
		{name: "crit that drops the thrall", outcome: rules.AttackOutcome{Hit: true, Crit: true, Natural: 20, HPAfter: 0}, impact: "sfx_crit_hit", delay: 500 + attackContact, fall: true},
		{name: "miss", outcome: rules.AttackOutcome{Natural: 3}, impact: "sfx_miss_whoosh", delay: 500 + attackContact - 150},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			list := sounds(t, AttackAudio(AttackResult{Seat: 2, Path: path, Outcome: tc.outcome}))
			if list[0].Name != "voicepack/2/attack_effort" || list[0].Target != "seat" || list[0].Seat != 2 {
				t.Fatalf("effort cue = %#v", list[0])
			}
			steps, ok := find(list, "sfx_forest_steps")
			if !ok || steps.DelayMS != 0 || steps.Target != "dm" {
				t.Fatalf("footsteps = %#v in %#v", steps, list)
			}
			impact, ok := find(list, tc.impact)
			if !ok || impact.DelayMS != tc.delay || impact.Target != "dm" || impact.Channel != vocab.SoundSFX {
				t.Fatalf("impact %q = %#v in %#v", tc.impact, impact, list)
			}
			if _, fell := find(list, "sfx_splash_collapse"); fell != tc.fall {
				t.Fatalf("collapse present = %v, want %v", fell, tc.fall)
			}
		})
	}
}

func TestAttackAudio_invalidSeatIsSilent(t *testing.T) {
	if got := AttackAudio(AttackResult{Seat: 0}); got != nil {
		t.Fatalf("got %#v", got)
	}
}

func TestAttackAudio_adjacentAttackHasNoFootsteps(t *testing.T) {
	list := sounds(t, AttackAudio(AttackResult{Seat: 1, Outcome: rules.AttackOutcome{Hit: true, Natural: 10, HPAfter: 4}}))
	if _, ok := find(list, "sfx_forest_steps"); ok {
		t.Fatalf("unexpected footsteps in %#v", list)
	}
	if AttackResolvedMS(AttackResult{Path: []Cell{{X: 1}}}) != WalkStepMS+attackResolved {
		t.Fatal("resolved time must follow the walk")
	}
}

func TestEnemyAudio_slamHurtAndDownFollowTheStartOffset(t *testing.T) {
	result := EnemyResult{TargetSeat: 1, Path: []Cell{{X: 3}}, ContactMS: 1200, Outcome: rules.AttackOutcome{AttackID: "slam-pc1", Hit: true, HPAfter: 0}}
	list := sounds(t, EnemyAudio(result, 2000))
	groan, _ := find(list, "sfx_thrall_groan")
	if groan.DelayMS != 2000 {
		t.Fatalf("groan = %#v", groan)
	}
	contact := 2000 + WalkStepMS + 1200
	for _, name := range []string{"sfx_thrall_slam", "voicepack/1/hurt", "sfx_phone_hurt"} {
		if got, ok := find(list, name); !ok || got.DelayMS != contact {
			t.Fatalf("%s = %#v in %#v", name, got, list)
		}
	}
	if hurt, _ := find(list, "sfx_phone_hurt"); hurt.Target != "seat" || hurt.Seat != 1 {
		t.Fatalf("phone hurt = %#v", hurt)
	}
	if down, ok := find(list, "sfx_down"); !ok || down.DelayMS != contact+fallAfterMS {
		t.Fatalf("down = %#v", down)
	}
	if EnemyResolvedMS(result, 2000) != contact+800 {
		t.Fatalf("resolved = %d", EnemyResolvedMS(result, 2000))
	}
}

func TestEnemyAudio_missNoReachAndInvalid(t *testing.T) {
	miss := sounds(t, EnemyAudio(EnemyResult{TargetSeat: 2, ContactMS: 1200, Outcome: rules.AttackOutcome{AttackID: "slam-pc2"}}, -5))
	if whoosh, ok := find(miss, "sfx_miss_whoosh"); !ok || whoosh.DelayMS != 1050 {
		t.Fatalf("miss = %#v", miss)
	}
	if _, ok := find(miss, "sfx_thrall_slam"); ok {
		t.Fatal("a miss must not slam")
	}
	walkOnly := sounds(t, EnemyAudio(EnemyResult{TargetSeat: 2, Path: []Cell{{X: 1}}}, 0))
	if len(walkOnly) != 2 {
		t.Fatalf("walk-only turn = %#v", walkOnly)
	}
	if got := EnemyAudio(EnemyResult{TargetSeat: 3}, 0); got != nil {
		t.Fatalf("invalid = %#v", got)
	}
}

func TestVictoryTurnAndFledAudio(t *testing.T) {
	victory := sounds(t, VictoryAudio(1, 2500))
	if victory[0].Name != "voicepack/1/victory" || victory[0].Seat != 1 || victory[1].Name != "STING_VICTORY" || victory[1].Channel != vocab.SoundMusic || victory[1].DelayMS != 2500 {
		t.Fatalf("victory = %#v", victory)
	}
	if VictoryAudio(0, 0) != nil || TurnAudio(3, 0) != nil {
		t.Fatal("invalid seats must be silent")
	}
	turn := sounds(t, TurnAudio(2, -1))
	if turn[0].Name != "sfx_your_turn" || turn[0].Seat != 2 || turn[0].DelayMS != 0 {
		t.Fatalf("turn = %#v", turn)
	}
	fled := sounds(t, FledAudio())
	if fled[0].Name != "STING_BELL_TOLL" || fled[0].Channel != vocab.SoundMusic {
		t.Fatalf("fled = %#v", fled)
	}
}
