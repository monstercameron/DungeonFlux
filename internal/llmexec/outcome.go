package llmexec

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// outcomePersona is Mother Vell's persona for the check outcome prompt.
const outcomePersona = "Gravelly, amused, protective of her regulars; evasive until persuaded."

// OutcomeExecutor streams Mother Vell's Persuasion outcome: the reveal, whose
// prompt carries the gated clue, or the refusal, whose prompt never does. The
// effect input is the player's line that led to the check.
type OutcomeExecutor struct {
	llm      ports.LLM
	clue     string
	template map[vocab.Role]prompts.Template
	Locales  LocaleSource
}

// NewOutcomeExecutor constructs an outcome executor. clue is the gated secret
// the reveal may speak; the refusal prompt is built without it.
func NewOutcomeExecutor(llm ports.LLM, clue string) *OutcomeExecutor {
	templates := make(map[vocab.Role]prompts.Template, 2)
	for _, role := range []vocab.Role{vocab.RoleNPCReveal, vocab.RoleNPCRefuse} {
		if template, err := prompts.TemplateFor(role); err == nil {
			templates[role] = template
		}
	}
	return &OutcomeExecutor{llm: llm, clue: strings.TrimSpace(clue), template: templates}
}

// Execute renders the reveal or refusal prompt and streams the reply.
func (e *OutcomeExecutor) Execute(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil || e.llm == nil {
		postFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
		return
	}
	request, err := e.request(effect)
	if err != nil {
		postFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	stream, err := e.llm.StreamText(ctx, WithLocale(request, e.Locales.ForRoom()))
	if err != nil {
		postFailure(ctx, scope, in, effect.UtteranceID, npcFailureKind(err))
		return
	}
	if stream == nil {
		postFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	defer stream.Close()
	e.consume(ctx, stream, effect, scope, in)
}

func (e *OutcomeExecutor) request(effect domain.StartLine) (ports.TextRequest, error) {
	template, ok := e.template[effect.Role]
	if !ok {
		return ports.TextRequest{}, errors.New("outcome: role must be npc_reveal or npc_refuse")
	}
	instruction := "The Persuasion check failed. Refuse firmly and reveal nothing about where the lamplighter was taken."
	if effect.Role == vocab.RoleNPCReveal {
		if e.clue == "" {
			return ports.TextRequest{}, errors.New("outcome: reveal requires the gated clue")
		}
		instruction = "The Persuasion check succeeded. Give in and tell them this secret plainly, in your own words: " + e.clue
	}
	utterance := strings.TrimSpace(effect.Input)
	if utterance == "" {
		utterance = "(presses her)"
	}
	text, err := template.Render(map[string]string{"persona": outcomePersona, "last_utterance": utterance, "clue": instruction})
	if err != nil {
		return ports.TextRequest{}, err
	}
	return ports.TextRequest{
		Meta:      ports.CallMeta{Role: effect.Role, UtteranceID: effect.UtteranceID},
		Messages:  []ports.Message{{Role: vocab.MsgUser, Text: text}},
		MaxTokens: 96,
	}, nil
}

func (e *OutcomeExecutor) consume(ctx context.Context, stream ports.TextStream, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	var text strings.Builder
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if validateErr := prompts.ValidateText(effect.Role, text.String()); validateErr != nil {
				postFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
				return
			}
			postNarration(ctx, scope, in, effect, "", text.String(), true)
			postEvent(ctx, scope, in, domain.LineDone{UtteranceID: effect.UtteranceID})
			return
		}
		if err != nil {
			if !isCanceled(ctx, err) {
				postFailure(ctx, scope, in, effect.UtteranceID, npcFailureKind(err))
			}
			return
		}
		if chunk == "" {
			continue
		}
		text.WriteString(chunk)
		postNarration(ctx, scope, in, effect, chunk, text.String(), false)
	}
}
