package i18n

import (
	"testing"
)

func TestVoiceLocale_Mapping(t *testing.T) {
	if got := STTLanguage("es"); got != "es-ES" {
		t.Fatalf("STTLanguage(es) = %q", got)
	}
	if got := STTLanguage("en"); got != "en-US" {
		t.Fatalf("STTLanguage(en) = %q", got)
	}
	if got := TTSVoice("es"); got != "es-narrator" {
		t.Fatalf("TTSVoice(es) = %q", got)
	}
	if got := TTSVoice("fr"); got != "en-narrator" {
		t.Fatalf("TTSVoice(fr) = %q, want english fallback", got)
	}
	cases := []struct {
		name, base, locale, want string
	}{
		{"default keeps bare id", "hook_intro", "en", "hook_intro"},
		{"spanish suffixes id", "hook_intro", "es", "hook_intro_es"},
		{"non-default tag suffixes id", "hook_intro", "fr", "hook_intro_fr"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CannedAssetID(tc.base, tc.locale); got != tc.want {
				t.Fatalf("CannedAssetID(%q, %q) = %q, want %q", tc.base, tc.locale, got, tc.want)
			}
		})
	}
}
