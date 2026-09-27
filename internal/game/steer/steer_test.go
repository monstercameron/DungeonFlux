package steer

import (
	"strconv"
	"testing"
)

func TestSelect_IntentControlsWhetherSteeringApplies(t *testing.T) {
	tests := []struct {
		name   string
		intent Intent
		want   Rung
	}{
		{name: "on funnel", intent: IntentOnFunnel, want: RungNone},
		{name: "curious off funnel", intent: IntentCurious, want: RungNone},
		{name: "drifting", intent: IntentDrifting, want: RungNone},
		{name: "stalled", intent: IntentStalled, want: RungLure},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := Select(Input{Intent: test.intent}).Rung
			if got != test.want {
				t.Fatalf("rung = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSelect_DriftingEscalatesOneRungPerBeat(t *testing.T) {
	tests := []struct {
		beats int
		want  Rung
	}{
		{beats: -1, want: RungNone},
		{beats: 0, want: RungNone},
		{beats: 1, want: RungLure},
		{beats: 2, want: RungPersonalHook},
		{beats: 3, want: RungClueRelocation},
		{beats: 4, want: RungConsequence},
		{beats: 5, want: RungWorldClosesIn},
		{beats: 99, want: RungWorldClosesIn},
	}
	for _, test := range tests {
		t.Run(testName(test.beats), func(t *testing.T) {
			got := Select(Input{ElapsedBeats: test.beats, Intent: IntentDrifting}).Rung
			if got != test.want {
				t.Fatalf("beats %d: rung = %v, want %v", test.beats, got, test.want)
			}
		})
	}
}

func TestRungString_UsesStableLogNames(t *testing.T) {
	tests := []struct {
		rung Rung
		want string
	}{
		{RungNone, "none"}, {RungLure, "lure"},
		{RungPersonalHook, "personal_hook"},
		{RungClueRelocation, "clue_relocation"},
		{RungConsequence, "consequence"},
		{RungWorldClosesIn, "world_closes_in"}, {Rung(99), "none"},
	}
	for _, test := range tests {
		if got := test.rung.String(); got != test.want {
			t.Errorf("rung %d: string = %q, want %q", test.rung, got, test.want)
		}
	}
}

func testName(beats int) string {
	return "beats_" + strconv.Itoa(beats)
}
