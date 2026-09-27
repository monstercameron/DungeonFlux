package domain

import "testing"

func TestViewDeepCopyDoesNotShareSlices(t *testing.T) {
	v := View{Seats: []SeatView{{Seat: 1}}, Preload: []string{"a"}, Battlefield: &BattlefieldView{Tokens: []TokenView{{ID: "t", Path: []Cell{{C: 1}}, Clips: map[string]AssetID{"idle": "clip"}}}, Highlights: []HighlightView{{Cells: []Cell{{R: 2}}}}}, Dice: &DiceView{Damage: &DamageView{Faces: []int{1, 2}}}}
	c := v.DeepCopy()
	c.Seats[0].Seat = 2
	c.Preload[0] = "b"
	c.Battlefield.Tokens[0].ID = "x"
	c.Battlefield.Tokens[0].Path[0].C = 9
	c.Battlefield.Tokens[0].Clips["idle"] = "changed"
	c.Battlefield.Highlights[0].Cells[0].R = 9
	c.Dice.Damage.Faces[0] = 9
	if v.Seats[0].Seat != 1 || v.Preload[0] != "a" || v.Battlefield.Tokens[0].ID != "t" || v.Battlefield.Tokens[0].Path[0].C != 1 || v.Battlefield.Tokens[0].Clips["idle"] != "clip" || v.Battlefield.Highlights[0].Cells[0].R != 2 || v.Dice.Damage.Faces[0] != 1 {
		t.Fatal("deep copy shares mutable state")
	}
}
