package i18n

import (
	"sort"
	"strings"
)

// Entry is one localizable string. Other holds the plural form used when
// Count is not 1; when empty, Text is used for every count.
type Entry struct {
	Text  string
	Other string
}

// Catalog maps locale tags to keyed string entries.
type Catalog struct {
	entries map[string]map[string]Entry
}

// NewCatalog builds a catalog from per-locale key tables.
func NewCatalog(locales map[string]map[string]Entry) Catalog {
	entries := make(map[string]map[string]Entry, len(locales))
	for locale, table := range locales {
		clone := make(map[string]Entry, len(table))
		for key, entry := range table {
			clone[key] = entry
		}
		entries[Normalize(locale)] = clone
	}
	return Catalog{entries: entries}
}

// Locales returns the sorted locale tags present in the catalog.
func (c Catalog) Locales() []string {
	out := make([]string, 0, len(c.entries))
	for locale := range c.entries {
		out = append(out, locale)
	}
	sort.Strings(out)
	return out
}

// Keys returns the sorted keys for one locale, or the default locale table
// when the locale is absent.
func (c Catalog) Keys(locale string) []string {
	table := c.entries[Normalize(locale)]
	if table == nil {
		table = c.entries[DefaultLocale]
	}
	out := make([]string, 0, len(table))
	for key := range table {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

// Has reports whether a locale table contains a key.
func (c Catalog) Has(locale, key string) bool {
	if table := c.entries[Normalize(locale)]; table != nil {
		_, ok := table[key]
		return ok
	}
	return false
}

// Lookup resolves a key for a locale with fallback to the default locale.
// It returns false only when no locale holds the key.
func (c Catalog) Lookup(locale, key string) (Entry, bool) {
	if table := c.entries[Normalize(locale)]; table != nil {
		if entry, ok := table[key]; ok {
			return entry, true
		}
	}
	if table := c.entries[DefaultLocale]; table != nil {
		if entry, ok := table[key]; ok {
			return entry, true
		}
	}
	return Entry{}, false
}

// T renders a key for a locale, substituting {arg} placeholders and
// selecting the plural form by count. Missing keys render the fallback, or
// the key itself when no fallback is given.
func (c Catalog) T(locale, key string, args map[string]string, count int, fallback string) string {
	entry, ok := c.Lookup(locale, key)
	if !ok {
		if fallback != "" {
			return Format(fallback, args)
		}
		return key
	}
	return Format(Plural(entry, count), args)
}

// Plural selects the singular or plural template for a count.
func Plural(entry Entry, count int) string {
	if count != 1 && entry.Other != "" {
		return entry.Other
	}
	return entry.Text
}

// Format substitutes {name} placeholders from args. Unknown placeholders
// are left intact so a missing argument is visible, not silent.
func Format(template string, args map[string]string) string {
	if len(args) == 0 || !strings.Contains(template, "{") {
		return template
	}
	pairs := make([]string, 0, len(args)*2)
	for key, value := range args {
		pairs = append(pairs, "{"+key+"}", value)
	}
	return strings.NewReplacer(pairs...).Replace(template)
}
