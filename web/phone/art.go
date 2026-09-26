package phone

import (
	"strings"
	"sync/atomic"
)

// ArtSource resolves a logical UI art name from the build-time manifest to a
// browser-loadable URL. The shell installs the gRPC-backed source at boot.
type ArtSource interface {
	ArtURL(name string) string
}

type artSourceHolder struct{ value atomic.Value }

var artSources artSourceHolder

// SetArtSource installs the resolver used by every phone screen.
func SetArtSource(source ArtSource) {
	if source == nil {
		source = noArtSource{}
	}
	artSources.value.Store(artSourceBox{source: source})
}

// ArtURL resolves a logical phone art name, or returns an empty string when
// the loader has not been installed yet.
func ArtURL(name string) string {
	box, ok := artSources.value.Load().(artSourceBox)
	if !ok || box.source == nil {
		return ""
	}
	return strings.TrimSpace(box.source.ArtURL(name))
}

type artSourceBox struct{ source ArtSource }

type noArtSource struct{}

func (noArtSource) ArtURL(string) string { return "" }

const (
	phoneBackgroundAsset = "ui/phone_bg"
	buttonPrimaryAsset   = "ui/button_primary"
	buttonSecondaryAsset = "ui/button_secondary"
	buttonDisabledAsset  = "ui/button_disabled"
	d20Asset             = "ui/d20"
	d20SuccessAsset      = "ui/d20_success"
	d20FailAsset         = "ui/d20_fail"
)

// moveArtAsset returns the medallion icon for a legal move.
func moveArtAsset(moveID string) string {
	assets := map[string]string{
		"attack":    "ui/icon_attack",
		"end_turn":  "ui/icon_end_turn",
		"leave":     "ui/icon_leave",
		"move":      "ui/icon_move",
		"persuade":  "ui/icon_persuade",
		"ready":     "ui/icon_ready",
		"step_away": "ui/icon_step_away",
		"talk_vell": "ui/icon_talk",
		"talk":      "ui/icon_talk",
		"roll_hero": "ui/icon_ready",
	}
	return assets[strings.ToLower(strings.TrimSpace(moveID))]
}

// speciesArtAsset returns the portrait tile for a creation species.
func speciesArtAsset(species string) string {
	name := strings.ToLower(strings.TrimSpace(species))
	if name == "" {
		return ""
	}
	return "ui/species_" + name
}

// classArtAsset returns the crest for classes with generated art.
func classArtAsset(className string) string {
	name := strings.ToLower(strings.TrimSpace(className))
	switch name {
	case "bard", "cleric", "paladin", "rogue", "wizard":
		return "ui/class_" + name
	default:
		return ""
	}
}

// statusArtAsset returns the status medallion for a known combat state.
func statusArtAsset(status string) string {
	name := strings.ToLower(strings.TrimSpace(status))
	switch {
	case strings.Contains(name, "bloodied"):
		return "ui/status_bloodied"
	case strings.Contains(name, "down"):
		return "ui/status_down"
	case strings.Contains(name, "spotlight"), strings.Contains(name, "focused"):
		return "ui/status_spotlight"
	default:
		return ""
	}
}

func artBackground(url string, fallback string) string {
	if url == "" {
		return fallback
	}
	return "url(\"" + url + "\")"
}

func artButtonStyle(asset, fallback string) map[string]string {
	style := map[string]string{
		"background-color":  fallback,
		"background-size":   "100% 100%",
		"background-repeat": "no-repeat",
	}
	if url := ArtURL(asset); url != "" {
		style["background-image"] = "url(\"" + url + "\")"
	}
	return style
}
