package prompts

import (
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestTemplateFor_RenderFixtures(t *testing.T) {
	fixtures := map[vocab.Role]map[string]string{
		vocab.RoleCharacterFlavor: {"species": "human", "gender": "woman", "class": "paladin", "background": "sailor", "hooks": "protect the crew"},
		vocab.RoleInterpret:       {"transcript": "I persuade her", "phase": "conversation", "glossary": "persuade means convince", "legal_moves": "persuade, step_away", "npc_last_line": "Not for free."},
		vocab.RoleOpening:         {"one_shot": "The Drowned Lantern", "characters": "Asha and Bram"},
		vocab.RoleNPCReply:        {"persona": "gravelly and protective", "public_facts": "keeper of the tavern", "conversation": "The rain is loud."},
		vocab.RoleNPCReveal:       {"persona": "gravelly", "last_utterance": "Please help us", "clue": "The lamplighter was dragged to the bell tower."},
		vocab.RoleNPCRefuse:       {"persona": "gravelly", "last_utterance": "Please help us", "clue": "No clue is supplied."},
		vocab.RoleStrangerLines:   {"hook": "find the missing lamp", "stranger": "a dripping courier", "clue": "the tower bell"},
		vocab.RoleCliffhanger:     {"brief": "the bell tolls", "characters": "Asha and Bram"},
		vocab.RoleCombatOutcomes:  {"characters": "Asha and Bram", "thrall": "a drowned thrall"},
	}
	for _, template := range AllTemplates() {
		t.Run(string(template.Role), func(t *testing.T) {
			text, err := template.Render(fixtures[template.Role])
			if err != nil {
				t.Fatalf("Render() error = %v", err)
			}
			if strings.Contains(text, "{{") {
				t.Fatalf("unexpanded slot in %q", text)
			}
			if !strings.Contains(text, string(template.Role)) && template.Role != vocab.RoleNPCReveal && template.Role != vocab.RoleNPCRefuse {
				t.Logf("role is represented by fixed prose: %s", template.Role)
			}
		})
	}
}

func TestTemplate_RenderRejectsBadInputs(t *testing.T) {
	template, _ := TemplateFor(vocab.RoleOpening)
	for name, values := range map[string]map[string]string{
		"missing": {"one_shot": "story"},
		"unknown": {"one_shot": "story", "characters": "heroes", "extra": "bad"},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := template.Render(values); err == nil {
				t.Fatal("Render() accepted invalid values")
			}
		})
	}
}

func TestValidateText(t *testing.T) {
	if err := ValidateText(vocab.RoleNPCReply, "A short answer."); err != nil {
		t.Fatal(err)
	}
	tooLong := strings.Repeat("word ", 31)
	if err := ValidateText(vocab.RoleNPCReply, tooLong); err == nil {
		t.Fatal("accepted over-cap text")
	}
	if err := ValidateText(vocab.RoleNPCReply, " "); err == nil {
		t.Fatal("accepted empty text")
	}
}

func TestValidateText_outcomeCutoffIsThirtyWords(t *testing.T) {
	cases := []struct {
		name    string
		role    vocab.Role
		words   int
		wantErr bool
	}{
		{name: "reveal at the prompt target", role: vocab.RoleNPCReveal, words: 25},
		{name: "reveal slightly over the target", role: vocab.RoleNPCReveal, words: 26},
		{name: "reveal at the cutoff", role: vocab.RoleNPCReveal, words: 30},
		{name: "reveal over the cutoff", role: vocab.RoleNPCReveal, words: 31, wantErr: true},
		{name: "refuse at the cutoff", role: vocab.RoleNPCRefuse, words: 30},
		{name: "refuse over the cutoff", role: vocab.RoleNPCRefuse, words: 31, wantErr: true},
		{name: "npc reply allows a little over the target", role: vocab.RoleNPCReply, words: 28},
		{name: "npc reply over the cutoff", role: vocab.RoleNPCReply, words: 31, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateText(tc.role, strings.TrimSpace(strings.Repeat("word ", tc.words)))
			if (err != nil) != tc.wantErr {
				t.Fatalf("ValidateText(%s, %d words) error = %v, want error %v", tc.role, tc.words, err, tc.wantErr)
			}
		})
	}
}

func TestTemplateFor_outcomePromptStillAsksForTwentyFive(t *testing.T) {
	for _, role := range []vocab.Role{vocab.RoleNPCReply, vocab.RoleNPCReveal, vocab.RoleNPCRefuse} {
		template, err := TemplateFor(role)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(template.User, "25 words or fewer") {
			t.Fatalf("%s prompt = %q, want the 25-word target", role, template.User)
		}
	}
}
