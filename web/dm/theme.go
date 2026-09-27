package dm

import "sort"

// ThemeToken names a CSS custom property used by the DM frame.
type ThemeToken string

const (
	// TokenInk is the deep background color.
	TokenInk ThemeToken = "--df-ink"
	// TokenPanel is the translucent panel color.
	TokenPanel ThemeToken = "--df-panel"
	// TokenParchment is the readable primary text color.
	TokenParchment ThemeToken = "--df-parchment"
	// TokenMuted is the secondary text color.
	TokenMuted ThemeToken = "--df-muted"
	// TokenGold is the primary accent color.
	TokenGold ThemeToken = "--df-gold"
	// TokenBlood is the damage and danger color.
	TokenBlood ThemeToken = "--df-blood"
	// TokenTeal is the success color.
	TokenTeal ThemeToken = "--df-teal"
)

// Theme contains the visual tokens for the DM display.
type Theme struct {
	Tokens map[ThemeToken]string
}

// DefaultTheme returns the dark-fantasy palette used by the DM frame.
func DefaultTheme() Theme {
	return Theme{Tokens: map[ThemeToken]string{
		TokenInk:       "#0f1117",
		TokenPanel:     "rgba(15, 17, 23, .86)",
		TokenParchment: "#efe6d2",
		TokenMuted:     "#a89f8c",
		TokenGold:      "#d9a441",
		TokenBlood:     "#b3372f",
		TokenTeal:      "#3aa39a",
	}}
}

// CSSVariables returns a stable, sorted declaration list for inline styles.
func (theme Theme) CSSVariables() []string {
	keys := make([]string, 0, len(theme.Tokens))
	for key := range theme.Tokens {
		keys = append(keys, string(key))
	}
	sort.Strings(keys)
	declarations := make([]string, 0, len(keys))
	for _, key := range keys {
		declarations = append(declarations, key+":"+theme.Tokens[ThemeToken(key)])
	}
	return declarations
}

// TransitionClass returns the class used while a phase layer changes.
func TransitionClass() string { return "df-dm-transition" }
