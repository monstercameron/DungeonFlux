package i18n

import (
	"testing"
)

func TestSwitcher_Set(t *testing.T) {
	switcher := NewSwitcher("en")
	if got := switcher.Set("es"); got != "es" {
		t.Fatalf("Set(es) = %q, want es", got)
	}
	if got := switcher.Set("fr"); got != "es" {
		t.Fatalf("Set(fr) = %q, want previous es", got)
	}
	if got := switcher.Set(""); got != "es" {
		t.Fatalf("Set(empty) = %q, want previous es", got)
	}
}

func TestDetectLocale_PriorityOrder(t *testing.T) {
	cases := []struct {
		name string
		tags []string
		want string
	}{
		{"first supported wins", []string{"es-MX", "en-US"}, "es"},
		{"skips unsupported", []string{"fr-FR", "en"}, "en"},
		{"skips blanks", []string{"", "  ", "es"}, "es"},
		{"empty is default", nil, DefaultLocale},
		{"all unsupported is default", []string{"fr", "de"}, DefaultLocale},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DetectLocale(tc.tags); got != tc.want {
				t.Fatalf("DetectLocale(%q) = %q, want %q", tc.tags, got, tc.want)
			}
		})
	}
}
