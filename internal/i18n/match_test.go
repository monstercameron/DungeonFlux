package i18n

import (
	"testing"
)

func TestMatchEnglish_TemplateArguments(t *testing.T) {
	key, args, ok := MatchEnglish("Target out of range (30 ft)")
	if !ok || key != "reason.OUT_OF_RANGE" || args["ft"] != "30" {
		t.Fatalf("match = %q %v %v", key, args, ok)
	}
	if _, _, ok := MatchEnglish("no such line anywhere"); ok {
		t.Fatal("unknown line matched")
	}
}
