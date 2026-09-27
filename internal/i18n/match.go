package i18n

import (
	"sort"
	"strings"
	"sync"
)

type templateMatcher struct {
	key      string
	segments []string
	names    []string
}

var englishMatchOnce sync.Once
var englishMatchKeys []string
var englishMatchTable map[string]Entry
var englishMatchTemplates []templateMatcher

func englishMatchData() ([]string, map[string]Entry, []templateMatcher) {
	englishMatchOnce.Do(func() {
		table := EnglishEntries()
		keys := make([]string, 0, len(table))
		for key := range table {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var matchers []templateMatcher
		for _, key := range keys {
			text := table[key].Text
			if !strings.Contains(text, "{") {
				continue
			}
			segments, names := splitTemplate(text)
			if !hasLiteral(segments) {
				continue
			}
			matchers = append(matchers, templateMatcher{key: key, segments: segments, names: names})
		}
		englishMatchKeys = keys
		englishMatchTable = table
		englishMatchTemplates = matchers
	})
	return englishMatchKeys, englishMatchTable, englishMatchTemplates
}

func hasLiteral(segments []string) bool {
	for _, segment := range segments {
		if segment != "" {
			return true
		}
	}
	return false
}

// MatchEnglish maps engine-rendered English text back to a catalog key plus
// extracted arguments. Templates with {placeholders} become matchers; exact
// entries match verbatim. It returns false when no English entry fits.
//
// Tie-break: keys are tried in sorted order and the first full match wins,
// so two templates with identical shapes always resolve to the same key.
func MatchEnglish(text string) (string, map[string]string, bool) {
	keys, table, matchers := englishMatchData()
	for _, key := range keys {
		if table[key].Text == text {
			return key, nil, true
		}
	}
	for _, matcher := range matchers {
		args := make(map[string]string, len(matcher.names))
		if matchSegments(matcher.segments, matcher.names, 0, text, args) {
			return matcher.key, args, true
		}
	}
	return "", nil, false
}

func splitTemplate(template string) ([]string, []string) {
	var segments []string
	var names []string
	for {
		start := strings.Index(template, "{")
		if start < 0 {
			return append(segments, template), names
		}
		end := strings.Index(template[start:], "}")
		if end < 0 {
			return append(segments, template), names
		}
		end += start
		segments = append(segments, template[:start])
		names = append(names, template[start+1:end])
		template = template[end+1:]
	}
}

func matchSegments(segments []string, names []string, index int, text string, args map[string]string) bool {
	literal := segments[index]
	if !strings.HasPrefix(text, literal) {
		return false
	}
	rest := text[len(literal):]
	if index == len(names) {
		return rest == ""
	}
	for end := 1; end <= len(rest); end++ {
		args[names[index]] = rest[:end]
		if matchSegments(segments, names, index+1, rest[end:], args) {
			return true
		}
	}
	delete(args, names[index])
	return false
}
