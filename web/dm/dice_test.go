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

func TestPresentDice_StatesHaveReadableVisuals(t *testing.T) {
	tests := []struct {
		name      string
		state     string
		d20       int32
		wantFace  string
		wantRoll  bool
		wantClass string
	}{
		{name: "offered", state: "offered", wantFace: "?", wantClass: "is-offered"},
		{name: "rolling", state: "rolling", wantFace: "…", wantRoll: true, wantClass: "is-rolling"},
		{name: "resolved", state: "resolved", d20: 17, wantFace: "17", wantClass: "is-resolved"},
		{name: "invalid result", state: "resolved", d20: 21, wantFace: "—", wantClass: "is-resolved"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := PresentDice(DiceView{State: test.state, D20: test.d20})
			if got.FaceLabel != test.wantFace || got.IsRolling != test.wantRoll || got.StateClass != test.wantClass {
				t.Fatalf("presentation = %#v", got)
			}
		})
	}
}

func TestPresentDice_ClassifiesOutcome(t *testing.T) {
	if got := PresentDice(DiceView{Outcome: "success"}); !got.IsSuccess || got.IsFailure {
		t.Fatalf("success presentation = %#v", got)
	}
	if got := PresentDice(DiceView{Outcome: "FAIL"}); !got.IsFailure || got.IsSuccess {
		t.Fatalf("failure presentation = %#v", got)
	}
}

func TestDiceArtName_ChoosesOutcomeVariant(t *testing.T) {
	tests := []struct {
		name    string
		view    DiceView
		wantArt string
	}{
		{name: "success", view: DiceView{Outcome: "success"}, wantArt: "ui/d20_success"},
		{name: "critical", view: DiceView{Crit: true}, wantArt: "ui/d20_success"},
		{name: "failure", view: DiceView{Outcome: "failure"}, wantArt: "ui/d20_fail"},
		{name: "default", view: DiceView{State: "rolling"}, wantArt: "ui/d20"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := diceArtName(test.view, PresentDice(test.view)); got != test.wantArt {
				t.Fatalf("dice art = %q, want %q", got, test.wantArt)
			}
		})
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
