package keyword

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestMatch_RecognizesKeywordsOnlyForLegalMoves(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		legal []vocab.MoveID
		want  vocab.MoveID
		match bool
	}{
		{name: "persuade", text: "I try to persuade her", legal: []vocab.MoveID{vocab.MovePersuade}, want: vocab.MovePersuade, match: true},
		{name: "convince with punctuation", text: "Can I CONVINCE her?", legal: []vocab.MoveID{vocab.MovePersuade}, want: vocab.MovePersuade, match: true},
		{name: "step away phrase", text: "Never mind... I step away.", legal: []vocab.MoveID{vocab.MoveStepAway}, want: vocab.MoveStepAway, match: true},
		{name: "later", text: "We'll do this later", legal: []vocab.MoveID{vocab.MoveStepAway}, want: vocab.MoveStepAway, match: true},
		{name: "illegal persuade", text: "I persuade the guard", legal: []vocab.MoveID{vocab.MoveStepAway}, match: false},
		{name: "leave is not implicit", text: "I leave", legal: []vocab.MoveID{vocab.MoveStepAway}, match: false},
		{name: "substring is not a keyword", text: "This is persuasive", legal: []vocab.MoveID{vocab.MovePersuade}, match: false},
		{name: "empty", text: "...", legal: []vocab.MoveID{vocab.MovePersuade}, match: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, matched := Match(test.text, test.legal)
			if matched != test.match || got != test.want {
				t.Fatalf("Match(%q, %v) = (%q, %t), want (%q, %t)", test.text, test.legal, got, matched, test.want, test.match)
			}
		})
	}
}

func TestMatch_LegalMoveOrderWinsWhenSeveralKeywordsMatch(t *testing.T) {
	got, matched := Match("persuade me, then never mind", []vocab.MoveID{vocab.MoveStepAway, vocab.MovePersuade})
	if !matched || got != vocab.MoveStepAway {
		t.Fatalf("Match returned (%q, %t), want (%q, true)", got, matched, vocab.MoveStepAway)
	}
}

func TestKeywords_ReturnsCopyAndEmptyForUnknown(t *testing.T) {
	got := Keywords(vocab.MovePersuade)
	if len(got) != 2 || got[0] != "persuade" || got[1] != "convince" {
		t.Fatalf("Keywords(persuade) = %v", got)
	}
	got[0] = "changed"
	if Keywords(vocab.MovePersuade)[0] != "persuade" {
		t.Fatal("Keywords returned shared mutable state")
	}
	if unknown := Keywords(vocab.MoveLeave); unknown != nil {
		t.Fatalf("Keywords(leave) = %v, want nil", unknown)
	}
}
