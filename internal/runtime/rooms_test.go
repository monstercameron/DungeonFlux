package runtime

import (
	"bytes"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestRoomState_Reset_repostsSeatsAndSplatWithStableSeed(t *testing.T) {
	state, err := NewRoomState([]byte("rehearsal"))
	if err != nil {
		t.Fatal(err)
	}
	state.Join(domain.Seat{ID: 2, PlayerNumber: 2, Token: "two"})
	state.Join(domain.Seat{ID: 1, PlayerNumber: 1, Token: "one"})
	state.SetSplat(domain.Report{ReportKind: vocab.ReportSplatReady, ID: "client"})
	first, err := state.Reset()
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.Reset()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first.Seed, second.Seed) || first.RunIndex != 1 || second.RunIndex != 2 {
		t.Fatalf("reset plans = %#v, %#v", first, second)
	}
	if len(first.Joins) != 2 || first.Joins[0].Seat != 1 || first.Joins[1].Seat != 2 {
		t.Fatalf("joins = %#v", first.Joins)
	}
	if first.Splat == nil || first.Splat.ReportKind != vocab.ReportSplatReady {
		t.Fatalf("splat = %#v", first.Splat)
	}
}

func TestRoomState_Reset_randomSeedsChange(t *testing.T) {
	state, err := NewRoomState(nil)
	if err != nil {
		t.Fatal(err)
	}
	first, err := state.Reset()
	if err != nil {
		t.Fatal(err)
	}
	second, err := state.Reset()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first.Seed, second.Seed) {
		t.Fatal("random reset seeds were equal")
	}
}

func TestRoomState_SetSplat_ignoresOtherReports(t *testing.T) {
	state, _ := NewRoomState([]byte("seed"))
	state.SetSplat(domain.Report{ReportKind: vocab.ReportPlaybackDone})
	plan, err := state.Reset()
	if err != nil {
		t.Fatal(err)
	}
	if plan.Splat != nil {
		t.Fatalf("unexpected splat report = %#v", plan.Splat)
	}
}
