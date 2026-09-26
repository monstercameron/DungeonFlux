package domain

import "testing"

func TestViewDeepCopyDoesNotShareSlices(t *testing.T) {
	v := View{Seats: []SeatView{{Seat: 1}}, Preload: []string{"a"}, Battlefield: &BattlefieldView{Tokens: []TokenView{{ID: "t"}}}, Dice: &DiceView{Damage: &DamageView{Faces: []int{1, 2}}}}
	c := v.DeepCopy()
	c.Seats[0].Seat = 2
	c.Preload[0] = "b"
	c.Battlefield.Tokens[0].ID = "x"
	c.Dice.Damage.Faces[0] = 9
	if v.Seats[0].Seat != 1 || v.Preload[0] != "a" || v.Battlefield.Tokens[0].ID != "t" || v.Dice.Damage.Faces[0] != 1 {
		t.Fatal("deep copy shares mutable state")
	}
}
