package i18n

import (
	"testing"
)

func TestLocalizeMove_LabelAndReason(t *testing.T) {
	catalog := Default()
	label, reason := LocalizeMove(catalog, "es", "persuade", "raw reason", nil)
	if label != "Persuadir +4 contra CD 10" {
		t.Fatalf("LocalizeMove es label = %q", label)
	}
	if reason != "raw reason" {
		t.Fatalf("LocalizeMove reason = %q, want passthrough", reason)
	}
	label, _ = LocalizeMove(catalog, "en", "missing_move", "", nil)
	if label != "move.missing_move" {
		t.Fatalf("LocalizeMove missing label = %q, want visible key fallback", label)
	}
}

func TestLocalizeReason_TemplateArgs(t *testing.T) {
	catalog := Default()
	cases := []struct {
		name, locale, code string
		args               map[string]string
		want               string
	}{
		{"english args", "en", "NOT_YOUR_TURN", map[string]string{"holder": "seat 1"}, "Waiting for seat 1"},
		{"spanish args", "es", "NOT_YOUR_TURN", map[string]string{"holder": "seat 1"}, "Esperando a seat 1"},
		{"no args", "en", "SPEAKING", nil, "Mother Vell is speaking"},
		{"unknown code falls back to key", "en", "NOPE", nil, "reason.NOPE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocalizeReason(catalog, tc.locale, tc.code, tc.args); got != tc.want {
				t.Fatalf("LocalizeReason(%q, %q) = %q, want %q", tc.locale, tc.code, got, tc.want)
			}
		})
	}
}
