package i18n

// Default returns the committed catalog with every supported locale.
func Default() Catalog {
	return NewCatalog(map[string]map[string]Entry{
		"en": EnglishEntries(),
		"es": SpanishEntries(),
	})
}

// LocalizeMove renders a move label and reason for a locale. Empty labels
// fall back to the catalog key so a missing entry is visible, not silent.
func LocalizeMove(c Catalog, locale, moveID, reason string, reasonArgs map[string]string) (string, string) {
	label := c.T(locale, MoveKey(moveID), nil, 0, "")
	return label, reason
}

// LocalizeReason renders a reason code with template arguments.
func LocalizeReason(c Catalog, locale, code string, args map[string]string) string {
	return c.T(locale, ReasonKey(code), args, 0, "")
}
