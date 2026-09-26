package domain

import (
	"encoding/json"
	"testing"
)

func TestEntitiesJSONRoundTrip(t *testing.T) {
	original := Seat{ID: 2, PlayerNumber: 2, Character: &Character{ID: "pc-2", Name: "Rin", HP: 9, MaxHP: 9, AC: 14, Portrait: "asset-1"}, Connected: true}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	var got Seat
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != original.ID || got.Character == nil || got.Character.Name != "Rin" || got.Character.Portrait != "asset-1" {
		t.Fatalf("round trip mismatch: %#v", got)
	}
}

func TestCellJSONUsesContractNames(t *testing.T) {
	data, err := json.Marshal(Cell{C: 3, R: 4})
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"c":3,"r":4}` {
		t.Fatalf("unexpected cell: %s", data)
	}
}
