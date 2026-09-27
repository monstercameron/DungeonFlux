package dm

import "testing"

func TestDefaultTheme_HasReadablePalette(t *testing.T) {
	theme := DefaultTheme()
	for _, token := range []ThemeToken{TokenInk, TokenPanel, TokenParchment, TokenMuted, TokenGold, TokenBlood, TokenTeal} {
		if theme.Tokens[token] == "" {
			t.Fatalf("missing token %q", token)
		}
	}
}

func TestThemeCSSVariables_IsStableAndComplete(t *testing.T) {
	declarations := DefaultTheme().CSSVariables()
	if len(declarations) != 7 {
		t.Fatalf("declaration count = %d", len(declarations))
	}
	for i := 1; i < len(declarations); i++ {
		if declarations[i-1] >= declarations[i] {
			t.Fatalf("declarations are not sorted: %#v", declarations)
		}
	}
}

func TestTransitionClass_IsStable(t *testing.T) {
	if got := TransitionClass(); got != "df-dm-transition" {
		t.Fatalf("transition class = %q", got)
	}
}
