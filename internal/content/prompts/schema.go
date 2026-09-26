package prompts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Schema is a strict JSON Schema document associated with a model role.
type Schema struct {
	Role vocab.Role
	JSON []byte
}

// SchemaFor returns the strict schema used for a structured model response.
func SchemaFor(role vocab.Role) (Schema, error) {
	var properties map[string]any
	var required []string
	switch role {
	case vocab.RoleCharacterFlavor:
		properties = map[string]any{"name": stringProperty(), "look": stringProperty(), "hook": stringProperty(), "pronouns": stringProperty()}
		required = []string{"name", "look", "hook", "pronouns"}
	case vocab.RoleInterpret:
		properties = map[string]any{"clean_text": stringProperty(), "kind": enumProperty("DIALOGUE", "MOVE"), "move_id": map[string]any{"anyOf": []any{stringProperty(), map[string]any{"type": "null"}}}}
		required = []string{"clean_text", "kind", "move_id"}
	case vocab.RoleStrangerLines:
		properties = textProperties("found", "not_found")
		required = []string{"found", "not_found"}
	case vocab.RoleCliffhanger:
		properties = textProperties("found_via_npc", "found_via_stranger")
		required = []string{"found_via_npc", "found_via_stranger"}
	case vocab.RoleCombatOutcomes:
		properties = textProperties("slain_by_seat1", "slain_by_seat2", "fled")
		required = []string{"slain_by_seat1", "slain_by_seat2", "fled"}
	default:
		return Schema{}, fmt.Errorf("role %q has no JSON schema", role)
	}
	document := map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	raw, err := json.Marshal(document)
	if err != nil {
		return Schema{}, fmt.Errorf("marshal %s schema: %w", role, err)
	}
	return Schema{Role: role, JSON: raw}, nil
}

// Validate checks syntax, required fields, exact field set, field types, and
// role-specific enum and word-cap constraints.
func (s Schema) Validate(raw []byte) error {
	if len(bytes.TrimSpace(raw)) == 0 {
		return fmt.Errorf("%s output is empty", s.Role)
	}
	var value map[string]json.RawMessage
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("%s output is not JSON: %w", s.Role, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("%s output has trailing JSON", s.Role)
	}
	expected := schemaFields(s.Role)
	if len(value) != len(expected) {
		return fmt.Errorf("%s output must contain exactly %d fields", s.Role, len(expected))
	}
	for key := range value {
		if !expected[key] {
			return fmt.Errorf("%s output has unknown field %q", s.Role, key)
		}
	}
	for key := range expected {
		rawField, ok := value[key]
		if !ok || bytes.Equal(bytes.TrimSpace(rawField), []byte("null")) && !(s.Role == vocab.RoleInterpret && key == "move_id") {
			return fmt.Errorf("%s output requires field %q", s.Role, key)
		}
		if s.Role == vocab.RoleInterpret && key == "move_id" && bytes.Equal(bytes.TrimSpace(rawField), []byte("null")) {
			continue
		}
		var text string
		if err := json.Unmarshal(rawField, &text); err != nil {
			return fmt.Errorf("%s field %q must be a string", s.Role, key)
		}
		if text == "" {
			return fmt.Errorf("%s field %q must not be empty", s.Role, key)
		}
		if err := ValidateText(s.Role, text); err != nil && key != "kind" && key != "move_id" {
			return err
		}
	}
	if s.Role == vocab.RoleInterpret {
		var kind string
		_ = json.Unmarshal(value["kind"], &kind)
		if kind != "DIALOGUE" && kind != "MOVE" {
			return fmt.Errorf("interpret kind %q is invalid", kind)
		}
	}
	return nil
}

func stringProperty() map[string]any { return map[string]any{"type": "string"} }
func enumProperty(values ...string) map[string]any {
	items := make([]any, len(values))
	for i, value := range values {
		items[i] = value
	}
	return map[string]any{"type": "string", "enum": items}
}
func textProperties(names ...string) map[string]any {
	result := make(map[string]any, len(names))
	for _, name := range names {
		result[name] = stringProperty()
	}
	return result
}
func schemaFields(role vocab.Role) map[string]bool {
	schema, err := SchemaFor(role)
	if err != nil {
		return map[string]bool{}
	}
	var document struct {
		Required []string `json:"required"`
	}
	_ = json.Unmarshal(schema.JSON, &document)
	result := make(map[string]bool, len(document.Required))
	for _, name := range document.Required {
		result[name] = true
	}
	return result
}

// String reports the role name, which is useful in diagnostics.
func (s Schema) String() string { return strings.TrimSpace(string(s.Role)) }
