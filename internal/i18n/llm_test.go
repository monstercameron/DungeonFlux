package i18n

import (
	"testing"
)

func TestPromptLanguage_Gating(t *testing.T) {
	cases := []struct {
		name, locale, want string
	}{
		{"english adds nothing", "en", ""},
		{"empty adds nothing", "", ""},
		{"unsupported adds nothing", "fr", ""},
		{"spanish names output language", "es", "Respond in Spanish (es-ES)."},
		{"region tag settles", "es-MX", "Respond in Spanish (es-ES)."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PromptLanguage(tc.locale); got != tc.want {
				t.Fatalf("PromptLanguage(%q) = %q, want %q", tc.locale, got, tc.want)
			}
		})
	}
}

func TestWithPromptLanguage_AppendsOnce(t *testing.T) {
	base := "Narrate the tavern."
	line := PromptLanguage("es")
	if got := WithPromptLanguage(base, "en"); got != base {
		t.Fatalf("WithPromptLanguage en changed prompt to %q", got)
	}
	want := base + "\n" + line
	if got := WithPromptLanguage(base, "es"); got != want {
		t.Fatalf("WithPromptLanguage es = %q, want %q", got, want)
	}
	if got := WithPromptLanguage(want, "es"); got != want {
		t.Fatalf("WithPromptLanguage es twice = %q, want no duplicate", got)
	}
	if got := WithPromptLanguage("", "es"); got != line {
		t.Fatalf("WithPromptLanguage empty = %q, want %q", got, line)
	}
}

func TestCacheKey_NamespacesLocale(t *testing.T) {
	if got := CacheKey("es", "hook"); got != "es|hook" {
		t.Fatalf("CacheKey(es) = %q", got)
	}
	if CacheKey("es", "hook") == CacheKey("en", "hook") {
		t.Fatal("es and en cache keys collide")
	}
	if got := CacheKey("es-MX", "hook"); got != CacheKey("es", "hook") {
		t.Fatalf("region tag not normalized: %q", got)
	}
}
