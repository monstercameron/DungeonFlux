// Package llmexec turns model effects into typed result events.
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

// NPCReplyExecutor streams an evasive Mother Vell reply from an LLM.
// Canned playback remains engine policy after LineFailed.
type NPCReplyExecutor struct {
	llm      ports.LLM
	template prompts.Template
}

// NpcReplyExecutor is the initialism-compatible name for NPCReplyExecutor.
type NpcReplyExecutor = NPCReplyExecutor

// NewNPCReplyExecutor constructs an executor using the fixed NPC reply prompt.
func NewNPCReplyExecutor(llm ports.LLM) *NPCReplyExecutor {
	template, _ := prompts.TemplateFor(vocab.RoleNPCReply)
	return &NPCReplyExecutor{llm: llm, template: template}
}

// NewNpcReplyExecutor constructs an executor using the fixed NPC reply prompt.
func NewNpcReplyExecutor(llm ports.LLM) *NpcReplyExecutor {
	return NewNPCReplyExecutor(llm)
}

// StartLine executes a spoken NPC reply and posts narration and completion
// events. The input is the conversation transcript supplied by the engine.
func (e *NPCReplyExecutor) StartLine(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
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
	stream, err := e.llm.StreamText(ctx, request)
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

// Execute is an alias suitable for runtime.Handle registration.
func (e *NPCReplyExecutor) Execute(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	e.StartLine(ctx, effect, scope, in)
}

func (e *NPCReplyExecutor) request(effect domain.StartLine) (ports.TextRequest, error) {
	values := map[string]string{
		"persona":      "Gravelly, amused, protective of her regulars; evasive until persuaded.",
		"public_facts": "Mother Vell keeps the Drowned Lantern. The lamplighter vanished, and the tavern stands beside the river.",
		"conversation": effect.Input,
	}
	text, err := e.template.Render(values)
	if err != nil {
		return ports.TextRequest{}, err
	}
	return ports.TextRequest{
		Meta:      ports.CallMeta{Role: vocab.RoleNPCReply, UtteranceID: effect.UtteranceID},
		Messages:  []ports.Message{{Role: vocab.MsgUser, Text: text}},
		MaxTokens: 96,
	}, nil
}

func (e *NPCReplyExecutor) consume(ctx context.Context, stream ports.TextStream, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	var text strings.Builder
	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if validateErr := prompts.ValidateText(vocab.RoleNPCReply, text.String()); validateErr != nil {
					postFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
					return
				}
				postEvent(ctx, scope, in, domain.LineDone{UtteranceID: effect.UtteranceID})
				return
			}
			if !isCanceled(ctx, err) {
				postFailure(ctx, scope, in, effect.UtteranceID, npcFailureKind(err))
			}
			return
		}
		if chunk == "" {
			continue
		}
		text.WriteString(chunk)
		postEvent(ctx, scope, in, domain.NarrationDelta{UtteranceID: effect.UtteranceID, Text: chunk})
	}
}

func npcFailureKind(err error) vocab.ErrKind {
	var callErr *ports.CallError
	if errors.As(err, &callErr) && callErr.Kind != "" {
		return callErr.Kind
	}
	return vocab.ErrUnavailable
}

func isCanceled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil
}

func postFailure(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID, kind vocab.ErrKind) {
	postEvent(ctx, scope, in, domain.LineFailed{UtteranceID: id, FailureKind: kind})
}

func postEvent(ctx context.Context, scope domain.Scope, in ports.Inbox, event domain.Event) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: event})
	}
}
