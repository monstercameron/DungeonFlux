package i18n

import "strings"

// DetectLocale picks the best supported locale from browser language tags
// in priority order, such as navigator.languages. Unsupported tags are
// skipped; an empty result is the default locale.
func DetectLocale(tags []string) string {
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		if Supported(trimmed) {
			return Normalize(trimmed)
		}
	}
	return DefaultLocale
}

// Switcher is the client language-switcher model: the active locale plus
// the ordered options rendered by the switcher control.
type Switcher struct {
	Active  string
	Options []string
}

// NewSwitcher builds a switcher settled on the active locale.
func NewSwitcher(active string) Switcher {
	tag := Settle(active, DefaultLocale)
	options := make([]string, len(SupportedLocales))
	copy(options, SupportedLocales)
	return Switcher{Active: tag, Options: options}
}

// Set changes the active locale, falling back for unsupported tags.
func (s *Switcher) Set(locale string) string {
	s.Active = Settle(locale, s.Active)
	return s.Active
}

// T renders a catalog key for the switcher's active locale.
func (s Switcher) T(c Catalog, key string, args map[string]string) string {
	return c.T(s.Active, key, args, 0, "")
}
