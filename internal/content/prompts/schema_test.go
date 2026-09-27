package prompts

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestSchema_ValidateSamples(t *testing.T) {
	samples := map[vocab.Role]string{
		vocab.RoleCharacterFlavor: `{"name":"Mira","look":"Rain cloak","hook":"Find my brother","pronouns":"she/her"}`,
		vocab.RoleInterpret:       `{"clean_text":"I persuade her","kind":"MOVE","move_id":"persuade"}`,
		vocab.RoleStrangerLines:   `{"found":"The clue followed me from the river.","not_found":"It followed me from the river."}`,
		vocab.RoleCliffhanger:     `{"found_via_npc":"The bell tolls.","found_via_stranger":"The letter opens."}`,
		vocab.RoleCombatOutcomes:  `{"slain_by_seat1":"The thrall falls.","slain_by_seat2":"The river takes its own.","fled":"The bell tolls."}`,
	}
	for role, sample := range samples {
		t.Run(string(role), func(t *testing.T) {
			schema, err := SchemaFor(role)
			if err != nil {
				t.Fatal(err)
			}
			if len(schema.JSON) == 0 {
				t.Fatal("schema is empty")
			}
			if err := schema.Validate([]byte(sample)); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestSchema_RejectsInvalidOutputs(t *testing.T) {
	schema, _ := SchemaFor(vocab.RoleCharacterFlavor)
	for name, sample := range map[string]string{
		"malformed":  `{"name":`,
		"missing":    `{"name":"Mira","look":"Rain cloak","hook":"Find my brother"}`,
		"extra":      `{"name":"Mira","look":"Rain cloak","hook":"Find my brother","pronouns":"she/her","x":"no"}`,
		"wrong_type": `{"name":"Mira","look":3,"hook":"Find my brother","pronouns":"she/her"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if err := schema.Validate([]byte(sample)); err == nil {
				t.Fatal("Validate() accepted invalid output")
			}
		})
	}
	interpret, _ := SchemaFor(vocab.RoleInterpret)
	if err := interpret.Validate([]byte(`{"clean_text":"hello","kind":"NOPE","move_id":null}`)); err == nil {
		t.Fatal("accepted invalid kind")
	}
	if err := interpret.Validate([]byte(`{"clean_text":"hello","kind":"DIALOGUE","move_id":3}`)); err == nil {
		t.Fatal("accepted invalid move_id")
	}
}

func TestSchemaFor_RejectsSpokenRole(t *testing.T) {
	if _, err := SchemaFor(vocab.RoleOpening); err == nil {
		t.Fatal("returned schema for plain text role")
	}
	if _, err := TemplateFor(vocab.Role("missing")); err == nil {
		t.Fatal("returned unknown template")
	}
}
