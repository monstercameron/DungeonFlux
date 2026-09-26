package i18n

import "strings"

// PromptLanguage returns the output-language instruction appended to model
// prompts for a locale. English adds no instruction; other locales name the
// required output language explicitly.
func PromptLanguage(locale string) string {
	switch Normalize(locale) {
	case "es":
		return "Respond in Spanish (es-ES)."
	default:
		return ""
	}
}

// WithPromptLanguage appends the locale instruction to a prompt template
// without duplicating it when already present.
func WithPromptLanguage(prompt, locale string) string {
	line := PromptLanguage(locale)
	if line == "" {
		return prompt
	}
	if strings.Contains(prompt, line) {
		return prompt
	}
	if strings.TrimSpace(prompt) == "" {
		return line
	}
	return strings.TrimRight(prompt, "\n") + "\n" + line
}

// CacheKey namespaces a model-cache key by locale so one language can never
// serve another's cached response.
func CacheKey(locale, key string) string {
	return Normalize(locale) + "|" + key
}
