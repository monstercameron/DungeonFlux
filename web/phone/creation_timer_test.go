package phone

import "testing"

func TestCreationTimerSecondsRoundsUpWithoutGoingNegative(t *testing.T) {
	for _, test := range []struct {
		remaining int64
		want      int64
	}{
		{remaining: -1, want: 0},
		{remaining: 0, want: 0},
		{remaining: 1, want: 1},
		{remaining: 1000, want: 1},
		{remaining: 1001, want: 2},
	} {
		if got := creationTimerSeconds(test.remaining); got != test.want {
			t.Fatalf("remaining %d = %d, want %d", test.remaining, got, test.want)
		}
	}
}

func TestCreationTimerLabelLocalizesPauseState(t *testing.T) {
	if got := creationTimerLabel("en", 22001, false); got != "Time to choose · 23s" {
		t.Fatalf("english label = %q", got)
	}
	if got := creationTimerLabel("es-MX", 22001, true); got != "Creación pausada · 23 s" {
		t.Fatalf("spanish paused label = %q", got)
	}
}

func TestCreationTimerRemainingAtAnchorsToServerSnapshot(t *testing.T) {
	tests := []struct {
		name        string
		remaining   int64
		anchor, now float64
		want        int64
	}{
		{name: "two seconds later", remaining: 30000, anchor: 1000, now: 3000, want: 28000},
		{name: "clock before anchor", remaining: 30000, anchor: 1000, now: 900, want: 30000},
		{name: "clamps at zero", remaining: 1000, anchor: 1000, now: 2500, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := creationTimerRemainingAt(test.remaining, test.anchor, test.now); got != test.want {
				t.Fatalf("remaining = %d, want %d", got, test.want)
			}
		})
	}
}
