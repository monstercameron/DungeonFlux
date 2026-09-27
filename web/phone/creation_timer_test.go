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
