package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

func TestDebugPatch_BuildsAndRenamesSeat(t *testing.T) {
	machine, err := New([]byte("debug-creation"))
	if err != nil {
		t.Fatal(err)
	}
	seat, err := machine.DebugPatch(1, map[string]string{
		"name": "Astra", "species": "human", "gender": "female", "class": "paladin",
	})
	if err != nil || !seat.Built || seat.Flavor.Name != "Astra" || seat.Build.HP <= 0 {
		t.Fatalf("seat=%#v err=%v", seat, err)
	}
	seat, err = machine.DebugPatch(domain.SeatID(1), map[string]string{"hp": "1"})
	if err != nil || seat.Build.HP != 1 {
		t.Fatalf("hp patch=%#v err=%v", seat, err)
	}
	if _, err := machine.DebugPatch(3, map[string]string{"name": "bad"}); err == nil {
		t.Fatal("invalid seat accepted")
	}
}

func TestDebugPatch_ValidatesIdentityAndValues(t *testing.T) {
	machine, err := New([]byte("debug-creation-errors"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fields := range []map[string]string{
		{"species": "unknown"}, {"gender": "unknown"}, {"class": "unknown"},
		{"hp": "1"}, {"cell": "0,0"},
	} {
		if _, err := machine.DebugPatch(1, fields); err == nil {
			t.Fatalf("fields %v accepted", fields)
		}
	}
	if _, err := machine.DebugPatch(1, map[string]string{"name": "  "}); err == nil {
		t.Fatal("blank name accepted")
	}
}
