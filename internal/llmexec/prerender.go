// Package llmexec turns content effects into validated model result events.
package llmexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// PrerenderTextExecutor generates and validates the text variants used by
// voice/out before they are converted into audio assets.
type PrerenderTextExecutor struct {
	llm     ports.LLM
	Locales LocaleSource
}

// NewPrerenderTextExecutor constructs a pre-render text executor.
func NewPrerenderTextExecutor(llm ports.LLM) *PrerenderTextExecutor {
	return &PrerenderTextExecutor{llm: llm}
}

// Execute generates the requested text set and posts its terminal event.
// Invalid model output is treated as a failed pre-render so the engine can
// select its canned fallback.
func (e *PrerenderTextExecutor) Execute(ctx context.Context, effect domain.PrerenderText, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	texts, err := e.generate(ctx, effect)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		post(ctx, in, scope, domain.PrerenderFailed{Set: effect.Set})
		return
	}
	if ctx.Err() != nil {
		return
	}
	post(ctx, in, scope, domain.PrerenderTextDone{Set: effect.Set, Texts: texts})
}

func (e *PrerenderTextExecutor) generate(ctx context.Context, effect domain.PrerenderText) ([]string, error) {
	if e == nil || e.llm == nil {
		return nil, errors.New("prerender text: LLM is required")
	}
	if effect.Set == "" || effect.Input == "" {
		return nil, errors.New("prerender text: set and input are required")
	}
	fields, err := prerenderFields(effect.Role, effect.Variants)
	if err != nil {
		return nil, err
	}
	template, err := prompts.TemplateFor(effect.Role)
	if err != nil {
		return nil, err
	}
	schema, err := prompts.SchemaFor(effect.Role)
	if err != nil {
		return nil, err
	}
	user, err := prerenderPrompt(template, effect.Input)
	if err != nil {
		return nil, err
	}
	request := ports.TextRequest{
		Meta:      ports.CallMeta{Role: effect.Role},
		Messages:  []ports.Message{{Role: vocab.MsgSystem, Text: template.System}, {Role: vocab.MsgUser, Text: user}},
		MaxTokens: maxTokens(template.MaxWords, len(fields)),
	}
	raw, err := e.llm.JSON(ctx, WithLocale(request, e.Locales.ForRoom()), ports.Schema{Name: string(effect.Role), JSON: schema.JSON})
	if err != nil {
		return nil, fmt.Errorf("generate %s: %w", effect.Role, err)
	}
	if err := schema.Validate(raw); err != nil {
		return nil, err
	}
	return extractFields(raw, fields)
}

func prerenderPrompt(template prompts.Template, input string) (string, error) {
	var values map[string]string
	if json.Unmarshal([]byte(input), &values) == nil && len(values) > 0 {
		rendered, err := template.Render(values)
		if err == nil {
			return rendered[len(template.System)+2:], nil
		}
	}
	if strings.TrimSpace(input) == "" {
		return "", errors.New("prerender text: input is empty")
	}
	return input, nil
}

func prerenderFields(role vocab.Role, variants int) ([]string, error) {
	var fields []string
	switch role {
	case vocab.RoleStrangerLines:
		fields = []string{"found", "not_found"}
	case vocab.RoleCliffhanger:
		fields = []string{"found_via_npc", "found_via_stranger"}
	case vocab.RoleCombatOutcomes:
		fields = []string{"slain_by_seat1", "slain_by_seat2", "fled"}
	default:
		return nil, fmt.Errorf("prerender text: role %q is not a pre-render set", role)
	}
	if variants != len(fields) {
		return nil, fmt.Errorf("prerender text: role %q requires %d variants, got %d", role, len(fields), variants)
	}
	return fields, nil
}

func extractFields(raw []byte, fields []string) ([]string, error) {
	var values map[string]string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("decode prerender text: %w", err)
	}
	texts := make([]string, 0, len(fields))
	for _, field := range fields {
		value, ok := values[field]
		if !ok || value == "" {
			return nil, fmt.Errorf("prerender text: missing %q", field)
		}
		texts = append(texts, value)
	}
	return texts, nil
}

func maxTokens(words, variants int) int {
	if words <= 0 {
		return 256
	}
	return words*variants*2 + 16
}
