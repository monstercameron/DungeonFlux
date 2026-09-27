package i18n

import (
	"testing"
)

func TestSettle_Branches(t *testing.T) {
	cases := []struct {
		name, requested, room, want string
	}{
		{"requested wins", "es", "en", "es"},
		{"region tag normalizes", "es-MX", "en", "es"},
		{"empty takes room", "", "es", "es"},
		{"unknown takes room", "fr", "es", "es"},
		{"unknown room falls to english", "fr", "de", "en"},
		{"empty room falls to english", "", "", "en"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Settle(tc.requested, tc.room); got != tc.want {
				t.Fatalf("Settle(%q, %q) = %q, want %q", tc.requested, tc.room, got, tc.want)
			}
		})
	}
}

func TestNormalize_SupportedBasics(t *testing.T) {
	if got := Normalize("ES-es"); got != "es" {
		t.Fatalf("Normalize = %q, want es", got)
	}
	if got := Normalize(""); got != DefaultLocale {
		t.Fatalf("Normalize empty = %q", got)
	}
	if !Supported("es") || !Supported("en") {
		t.Fatal("en and es must be supported")
	}
	if Supported("fr") {
		t.Fatal("fr must not be supported")
	}
}
