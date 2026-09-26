package i18n

import "strings"

// DefaultLocale is the fallback language for every lookup.
const DefaultLocale = "en"

// SupportedLocales lists every locale with a committed catalog, in
// preference order.
var SupportedLocales = []string{"en", "es"}

// Normalize returns the canonical BCP-47-ish tag for a locale hint.
// "ES-es", "es_ES", and " es " all become "es". Unknown or empty hints
// become the default locale.
func Normalize(hint string) string {
	tag := strings.ToLower(strings.TrimSpace(hint))
	if tag == "" {
		return DefaultLocale
	}
	if i := strings.IndexAny(tag, "-_"); i >= 0 {
		tag = tag[:i]
	}
	if tag == "" {
		return DefaultLocale
	}
	return tag
}

// Supported reports whether a locale has a committed catalog.
func Supported(locale string) bool {
	tag := Normalize(locale)
	for _, item := range SupportedLocales {
		if item == tag {
			return true
		}
	}
	return false
}

// Settle resolves a requested locale against the supported set.
// Empty and unknown tags fall back to the room default, then to English.
func Settle(requested, roomDefault string) string {
	if strings.TrimSpace(requested) != "" && Supported(requested) {
		return Normalize(requested)
	}
	if Supported(roomDefault) {
		return Normalize(roomDefault)
	}
	return DefaultLocale
}
