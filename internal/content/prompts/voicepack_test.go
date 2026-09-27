package prompts

import (
	"strings"
	"testing"
)

func TestVoicePackPrompt_ContainsIdentityAndCue(t *testing.T) {
	prompt := VoicePackPrompt("elf", "female", "wizard", "river debt", VoiceCueSpellCast)
	for _, want := range []string{"elf", "female", "wizard", "river debt", "signature spell", "no words"} {
		if !strings.Contains(strings.ToLower(prompt), want) {
			t.Fatalf("prompt missing %q: %s", want, prompt)
		}
	}
}

func TestVoicePackCuesAndDurations_AreStableAndBounded(t *testing.T) {
	cues := VoicePackCues()
	if len(cues) != 6 {
		t.Fatalf("cue count=%d", len(cues))
	}
	seen := make(map[VoiceCue]bool, len(cues))
	for _, cue := range cues {
		if seen[cue] {
			t.Fatalf("duplicate cue %q", cue)
		}
		seen[cue] = true
		duration := VoicePackDurationSeconds(cue)
		if duration < 0.4 || duration > 1.5 {
			t.Fatalf("cue %q duration=%v", cue, duration)
		}
	}
	if got := VoicePackDurationSeconds("unknown"); got < 0.4 || got > 1.5 {
		t.Fatalf("unknown duration=%v", got)
	}
}

func TestVoicePackPrompt_EmptyIdentityUsesSafePlaceholders(t *testing.T) {
	prompt := VoicePackPrompt("", "", "", "", VoiceCueHurt)
	if strings.Count(prompt, "unspecified") != 4 || !strings.Contains(prompt, "pained grunt") {
		t.Fatalf("prompt=%s", prompt)
	}
}
