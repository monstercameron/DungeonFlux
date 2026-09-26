package audio

import (
	"math"
	"testing"
)

func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestLineClock_Place(t *testing.T) {
	lead := JitterLead.Seconds()
	tests := []struct {
		name   string
		chunks [][3]float64 // offset, duration, now
		want   []float64
	}{
		{"on-time chunks play back to back", [][3]float64{{lead, .1, 10}, {lead + .1, .1, 10.1}, {lead + .2, .1, 10.2}}, []float64{10 + lead, 10.25, 10.35}},
		{"late chunk shifts the rest of the line", [][3]float64{{lead, .1, 10}, {lead + .1, .1, 10.5}, {lead + .2, .1, 10.55}}, []float64{10 + lead, 10.5 + lead, 10.5 + lead + .1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var clock lineClock
			for i, c := range tc.chunks {
				if got := clock.place("line", c[0], c[1], c[2]); !near(got, tc.want[i]) {
					t.Fatalf("chunk %d at %v, want %v", i, got, tc.want[i])
				}
			}
		})
	}
}

func TestLineClock_FinishedLineIsNotExpiredUntilItsTailPlays(t *testing.T) {
	var clock lineClock
	clock.place("line", .15, .1, 10)
	clock.place("line", .25, .1, 10.1)
	if got := clock.expired(20); len(got) != 0 {
		t.Fatalf("open line expired: %v", got)
	}
	clock.finish("line")
	end, ok := clock.endOf("line")
	if !ok || !near(end, 10.35) {
		t.Fatalf("end = %v, %v; want 10.35", end, ok)
	}
	if got := clock.expired(10.3); len(got) != 0 {
		t.Fatalf("line expired while its tail was still queued: %v", got)
	}
	if got := clock.expired(10.4); len(got) != 1 || got[0] != "line" {
		t.Fatalf("expired = %v, want [line]", got)
	}
	clock.drop("line")
	if _, ok := clock.endOf("line"); ok {
		t.Fatal("dropped line kept its end time")
	}
}

func TestLineClock_NewUtteranceAnchorsIndependently(t *testing.T) {
	var clock lineClock
	clock.place("one", .15, .1, 10)
	clock.finish("one")
	if got := clock.place("two", .15, .1, 12); !near(got, 12.15) {
		t.Fatalf("second line at %v, want 12.15", got)
	}
	if got := clock.place("one", .15, .1, 13); !near(got, 13.15) {
		t.Fatalf("reused finished id at %v, want a fresh anchor 13.15", got)
	}
}
