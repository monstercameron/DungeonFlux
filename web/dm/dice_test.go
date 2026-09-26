package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestDiceViewFromProto_CopiesRollAndDamage(t *testing.T) {
	dice := &dungeonfluxv1.Dice{State: dungeonfluxv1.DiceState_DICE_STATE_RESOLVED, D20: 17, Modifier: 4, Dc: 10, Outcome: "success", Kind: dungeonfluxv1.DiceKind_DICE_KIND_CHECK, Damage: &dungeonfluxv1.Damage{Faces: []int32{6, 3}, Total: 9, Type: "slashing"}}
	got := DiceViewFromProto(dice)
	if got.State != "resolved" || got.Kind != "check" || got.D20 != 17 || got.Damage.Total != 9 {
		t.Fatalf("dice view = %#v", got)
	}
	dice.Damage.Faces[0] = 20
	if got.Damage.Faces[0] != 6 {
		t.Fatal("damage faces share wire storage")
	}
}

func TestDiceViewFromProto_NilAndDMView(t *testing.T) {
	if got := DiceViewFromProto(nil); got != (DiceView{}) {
		t.Fatalf("nil dice = %#v", got)
	}
	view := &dungeonfluxv1.DMView{Dice: &dungeonfluxv1.Dice{State: dungeonfluxv1.DiceState_DICE_STATE_OFFERED}}
	if got := DiceViewFromDMView(view); got.State != "offered" {
		t.Fatalf("dm dice = %#v", got)
	}
}

func TestDiceViewFromProto_UnknownEnumsAreEmpty(t *testing.T) {
	got := DiceViewFromProto(&dungeonfluxv1.Dice{})
	if got.State != "" || got.Kind != "" {
		t.Fatalf("unknown values = %#v", got)
	}
}

func TestTimerViewFromProto_CopiesFields(t *testing.T) {
	got := TimerViewFromProto(&dungeonfluxv1.Timer{Seat: "2", RemainingMs: 1200, TotalMs: 3000, Frozen: true})
	if got.Seat != "2" || got.RemainingMS != 1200 || got.TotalMS != 3000 || !got.Frozen {
		t.Fatalf("timer view = %#v", got)
	}
}

func TestTimerViewFromProto_NilIsEmpty(t *testing.T) {
	if got := TimerViewFromProto(nil); got != (TimerView{}) {
		t.Fatalf("nil timer = %#v", got)
	}
}

func TestTimerViewFromDMView_ProjectsTimer(t *testing.T) {
	view := &dungeonfluxv1.DMView{TurnTimer: &dungeonfluxv1.Timer{RemainingMs: 500}}
	if got := TimerViewFromDMView(view); got.RemainingMS != 500 {
		t.Fatalf("dm timer = %#v", got)
	}
}
