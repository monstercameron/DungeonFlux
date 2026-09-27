package phone

import "strings"

// PhoneMode identifies the icon and label raised in the center tab.
type PhoneMode string

const (
	// PhoneModePlay is the conversation mode.
	PhoneModePlay PhoneMode = "play"
	// PhoneModeExplore is the scene and investigation mode.
	PhoneModeExplore PhoneMode = "explore"
	// PhoneModeCombat is the combat action mode.
	PhoneModeCombat PhoneMode = "combat"
)

// PhoneTabID identifies one destination in the phone's five-tab frame.
type PhoneTabID string

const (
	// PhoneTabCharacter opens the character sheet.
	PhoneTabCharacter PhoneTabID = "character"
	// PhoneTabJournal opens the narration transcript.
	PhoneTabJournal PhoneTabID = "journal"
	// PhoneTabPlay opens the current phase.
	PhoneTabPlay PhoneTabID = "play"
	// PhoneTabMap opens the scene or combat map.
	PhoneTabMap PhoneTabID = "map"
	// PhoneTabMenu opens settings and the leave-table action.
	PhoneTabMenu PhoneTabID = "menu"
)

// PhoneTab is the render-safe label and icon for one frame tab.
type PhoneTab struct {
	ID       PhoneTabID
	Label    string
	Icon     string
	Active   bool
	Raised   bool
	Disabled bool
}

// ChoiceRowModel contains the content and state for an ornate action row.
type ChoiceRowModel struct {
	ID          string
	Label       string
	Reason      string
	Icon        string
	Enabled     bool
	Highlighted bool
}

// ResultBannerModel contains the state shown after a check resolves.
type ResultBannerModel struct {
	Success bool
	Message string
}

// StatValue is one ability score and its signed modifier.
type StatValue struct {
	Name     string
	Score    int32
	Modifier int32
}

// FrameTabs returns the stable five-tab order used by every phone screen.
func FrameTabs(locale string, active PhoneTabID, mode PhoneMode) []PhoneTab {
	if active == "" {
		active = PhoneTabPlay
	}
	return []PhoneTab{
		{ID: PhoneTabCharacter, Label: T(locale, "tabs.character", nil), Icon: "♙", Active: active == PhoneTabCharacter},
		{ID: PhoneTabJournal, Label: T(locale, "tabs.journal", nil), Icon: "▤", Active: active == PhoneTabJournal},
		{ID: PhoneTabPlay, Label: T(locale, "tabs.play", nil), Icon: modeIcon(mode), Active: active == PhoneTabPlay, Raised: true},
		{ID: PhoneTabMap, Label: T(locale, "tabs.map", nil), Icon: "⌖", Active: active == PhoneTabMap},
		{ID: PhoneTabMenu, Label: T(locale, "tabs.menu", nil), Icon: "☰", Active: active == PhoneTabMenu},
	}
}

// NormalizePhoneMode returns the safe default for an unknown mode.
func NormalizePhoneMode(mode PhoneMode) PhoneMode {
	switch mode {
	case PhoneModeExplore, PhoneModeCombat, PhoneModePlay:
		return mode
	default:
		return PhoneModePlay
	}
}

// ModeLabel returns the concise title used by the center mode tab.
func ModeLabel(mode PhoneMode) string {
	switch NormalizePhoneMode(mode) {
	case PhoneModeExplore:
		return "Explore"
	case PhoneModeCombat:
		return "Fight"
	default:
		return "Play"
	}
}

// ChoiceRowReason returns a stable accessible reason for a disabled choice.
func ChoiceRowReason(row ChoiceRowModel) string {
	if row.Enabled || strings.TrimSpace(row.Reason) != "" {
		return strings.TrimSpace(row.Reason)
	}
	return "Unavailable right now"
}

// HPPercent returns a clamped percentage for a character's health bar.
func HPPercent(hp, maximum int32) int32 {
	if maximum <= 0 || hp <= 0 {
		return 0
	}
	if hp >= maximum {
		return 100
	}
	return hp * 100 / maximum
}

func modeIcon(mode PhoneMode) string {
	switch NormalizePhoneMode(mode) {
	case PhoneModeExplore:
		return "✥"
	case PhoneModeCombat:
		return "⚔"
	default:
		return "●"
	}
}
