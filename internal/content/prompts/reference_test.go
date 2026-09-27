package prompts

import (
	"strings"
	"testing"
)

func TestReferencePrompt_PreservesIdentityInputsAndConstraints(t *testing.T) {
	text := ReferencePrompt("elf", "female", "rogue", "Mira", "scarred river debt")
	for _, want := range []string{"elf", "female", "rogue", "Mira", "scarred river debt", "#0f1117", "exactly four equal panels", "No text"} {
		if !strings.Contains(text, want) {
			t.Fatalf("prompt missing %q: %s", want, text)
		}
	}
}
