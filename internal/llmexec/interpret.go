// Package llmexec executes structured language-model effects for the room.
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

// InterpretConfig supplies the model and the fixed context used by the
// interpret prompt. The engine supplies the legal moves on each effect.
type InterpretConfig struct {
	LLM      ports.LLM
	Phase    vocab.StateID
	Glossary string
}

// InterpretExecutor classifies a transcript into dialogue or a proposed move.
type InterpretExecutor struct {
	llm      ports.LLM
	phase    vocab.StateID
	glossary string
}

// NewInterpretExecutor constructs an interpret executor.
func NewInterpretExecutor(config InterpretConfig) *InterpretExecutor {
	phase := config.Phase
	if phase == "" {
		phase = vocab.StateConversation
	}
	glossary := strings.TrimSpace(config.Glossary)
	if glossary == "" {
		glossary = "persuade means convince; step_away means leave the conversation"
	}
	return &InterpretExecutor{llm: config.LLM, phase: phase, glossary: glossary}
}

// Execute classifies effect.Transcript and posts one terminal result event.
// Cancellation is terminal for the work item but is intentionally not posted
// because the runner has already discarded the effect's scope.
func (e *InterpretExecutor) Execute(ctx context.Context, effect domain.Interpret, scope domain.Scope, in ports.Inbox) {
	if ctx.Err() != nil {
		return
	}
	result, err := e.interpret(ctx, effect)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		interpretPost(ctx, in, domain.Envelope{Scope: scope, Event: domain.InterpretFailed{UtteranceID: effect.UtteranceID}})
		return
	}
	interpretPost(ctx, in, domain.Envelope{Scope: scope, Event: result})
}

func (e *InterpretExecutor) interpret(ctx context.Context, effect domain.Interpret) (domain.Interpreted, error) {
	if e == nil || e.llm == nil {
		return domain.Interpreted{}, errors.New("interpret: LLM is required")
	}
	template, err := prompts.TemplateFor(vocab.RoleInterpret)
	if err != nil {
		return domain.Interpreted{}, fmt.Errorf("load interpret prompt: %w", err)
	}
	schema, err := prompts.SchemaFor(vocab.RoleInterpret)
	if err != nil {
		return domain.Interpreted{}, fmt.Errorf("load interpret schema: %w", err)
	}
	rendered, err := template.Render(map[string]string{
		"transcript":    strings.TrimSpace(effect.Transcript),
		"phase":         string(e.phase),
		"glossary":      e.glossary,
		"legal_moves":   joinMoves(effect.Moves),
		"npc_last_line": nonEmpty(effect.NPCLastLine, "(none)"),
	})
	if err != nil {
		return domain.Interpreted{}, fmt.Errorf("render interpret prompt: %w", err)
	}
	user := strings.TrimPrefix(rendered, template.System+"\n\n")
	answer, err := e.llm.JSON(ctx, ports.TextRequest{
		Meta: ports.CallMeta{
			Role:        vocab.RoleInterpret,
			Phase:       e.phase,
			Seat:        effect.Seat,
			UtteranceID: effect.UtteranceID,
		},
		Messages:  []ports.Message{{Role: vocab.MsgSystem, Text: template.System}, {Role: vocab.MsgUser, Text: user}},
		MaxTokens: 128,
	}, ports.Schema{Name: string(vocab.RoleInterpret), JSON: schema.JSON})
	if err != nil {
		return domain.Interpreted{}, fmt.Errorf("interpret model call: %w", err)
	}
	if err := schema.Validate(answer); err != nil {
		return domain.Interpreted{}, fmt.Errorf("validate interpret output: %w", err)
	}
	var decoded interpretOutput
	if err := json.Unmarshal(answer, &decoded); err != nil {
		return domain.Interpreted{}, fmt.Errorf("decode interpret output: %w", err)
	}
	return domain.Interpreted{
		UtteranceID:        effect.UtteranceID,
		CleanText:          decoded.CleanText,
		InterpretationKind: decoded.Kind,
		Move:               moveID(decoded.MoveID),
	}, nil
}

type interpretOutput struct {
	CleanText string  `json:"clean_text"`
	Kind      string  `json:"kind"`
	MoveID    *string `json:"move_id"`
}

func joinMoves(moves []vocab.MoveID) string {
	if len(moves) == 0 {
		return "none"
	}
	values := make([]string, len(moves))
	for i, move := range moves {
		values[i] = string(move)
	}
	return strings.Join(values, ", ")
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func moveID(value *string) vocab.MoveID {
	if value == nil {
		return ""
	}
	return vocab.MoveID(*value)
}

func interpretPost(ctx context.Context, in ports.Inbox, envelope domain.Envelope) {
	if in != nil {
		in.Post(ctx, envelope)
	}
}
