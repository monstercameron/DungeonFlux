package i18n

// STTLanguage returns the BCP-47 language hint sent with speech-to-text
// audio for a locale.
func STTLanguage(locale string) string {
	switch Normalize(locale) {
	case "es":
		return "es-ES"
	default:
		return "en-US"
	}
}

// TTSVoice returns the configured text-to-speech voice for a locale.
// Voices are logical names resolved to vendor voice IDs at wiring time.
func TTSVoice(locale string) string {
	switch Normalize(locale) {
	case "es":
		return "es-narrator"
	default:
		return "en-narrator"
	}
}

// CannedAssetID resolves the per-locale audio asset for a canned line.
// The default locale keeps the bare ID; other locales use an ID suffix so
// existing assets keep working while translations land.
func CannedAssetID(baseID, locale string) string {
	tag := Normalize(locale)
	if tag == DefaultLocale || tag == "" {
		return baseID
	}
	return baseID + "_" + tag
}
