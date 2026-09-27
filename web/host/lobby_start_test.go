package host

import (
	"strings"
	"testing"
)

func TestLobbyStart_EligibilityAndVisibleHint(t *testing.T) {
	for _, tc := range []struct {
		name     string
		modify   func(*hostSnapshot)
		canStart bool
		hint     string
	}{
		{"both ready", func(s *hostSnapshot) {}, true, "Start the adventure"},
		{"connecting", func(s *hostSnapshot) { s.Connected = false }, false, "Connecting"},
		{"one joined", func(s *hostSnapshot) { s.SeatsJoined = 1 }, false, "both players to join"},
		{"one ready", func(s *hostSnapshot) { s.SeatsReady = 1 }, false, "choose Ready"},
		{"paused", func(s *hostSnapshot) { s.Paused = true }, false, "Resume"},
		{"already started", func(s *hostSnapshot) { s.Phase = "creation" }, false, ""},
		{"spanish ready", func(s *hostSnapshot) { s.Locale = "es" }, true, "Inicia la aventura"},
		{"spanish waiting", func(s *hostSnapshot) { s.Locale = "es"; s.SeatsReady = 0 }, false, "pulsen Listo"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := hostSnapshot{Connected: true, Phase: "lobby", SeatsJoined: 2, SeatsReady: 2, SeatsTotal: 2}
			tc.modify(&snapshot)
			if got := canStart(snapshot); got != tc.canStart {
				t.Fatalf("canStart = %v, want %v", got, tc.canStart)
			}
			got := lobbyStartHint(snapshot)
			if (tc.hint == "" && got != "") || !strings.Contains(got, tc.hint) {
				t.Fatalf("hint = %q, want %q", got, tc.hint)
			}
		})
	}
}
