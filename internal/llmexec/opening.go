// Package llmexec turns content prompts into engine result events.
package llmexec

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/content/prompts"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// OpeningExecutor streams an opening narration and publishes its deltas.
type OpeningExecutor struct {
	llm ports.LLM
}

// NewOpeningExecutor constructs an opening executor around an LLM chain.
func NewOpeningExecutor(llm ports.LLM) *OpeningExecutor {
	return &OpeningExecutor{llm: llm}
}

// StartLine executes an opening narration. It is an alias suitable for
// runtime registration alongside the other spoken-line executors.
func (e *OpeningExecutor) StartLine(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	e.Execute(ctx, effect, scope, in)
}

// Execute renders and streams an opening. The effect input is a JSON object
// with one_shot and characters fields, or the already-rendered prompt text.
func (e *OpeningExecutor) Execute(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil || e.llm == nil || effect.Role != vocab.RoleOpening {
		postLineFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
		return
	}
	request, err := openingRequest(effect.Input)
	if err != nil {
		postLineFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	template, err := prompts.TemplateFor(vocab.RoleOpening)
	if err != nil {
		postLineFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	rendered, err := template.Render(map[string]string{
		"one_shot":   request.OneShot,
		"characters": request.Characters,
	})
	if err != nil {
		postLineFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	stream, err := e.llm.StreamText(ctx, ports.TextRequest{
		Meta: ports.CallMeta{Role: vocab.RoleOpening, Phase: vocab.StateOpening, UtteranceID: effect.UtteranceID},
		Messages: []ports.Message{
			{Role: vocab.MsgSystem, Text: template.System},
			{Role: vocab.MsgUser, Text: rendered[len(template.System)+2:]},
		},
		MaxTokens: template.MaxWords * 2,
	})
	if err != nil || stream == nil {
		postLineFailure(ctx, scope, in, effect.UtteranceID, failureKind(err))
		return
	}
	defer stream.Close()
	var text strings.Builder
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			if err := prompts.ValidateText(vocab.RoleOpening, text.String()); err != nil {
				postLineFailure(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
				return
			}
			post(ctx, in, scope, domain.LineDone{UtteranceID: effect.UtteranceID})
			return
		}
		if recvErr != nil {
			if ctx.Err() == nil {
				postLineFailure(ctx, scope, in, effect.UtteranceID, failureKind(recvErr))
			}
			return
		}
		if strings.TrimSpace(chunk) == "" {
			continue
		}
		text.WriteString(chunk)
		post(ctx, in, scope, domain.NarrationDelta{UtteranceID: effect.UtteranceID, Text: chunk})
	}
}

// OpeningRequest supplies the dynamic values for the opening prompt.
type OpeningRequest struct {
	OneShot    string `json:"one_shot"`
	Characters string `json:"characters"`
}

func openingRequest(input string) (OpeningRequest, error) {
	var request OpeningRequest
	if err := json.Unmarshal([]byte(input), &request); err == nil {
		if strings.TrimSpace(request.OneShot) != "" && strings.TrimSpace(request.Characters) != "" {
			return request, nil
		}
	}
	if strings.TrimSpace(input) == "" {
		return OpeningRequest{}, errors.New("opening input is empty")
	}
	return OpeningRequest{OneShot: input, Characters: "the two heroes"}, nil
}

// CharacterFlavorExecutor validates a character flavor response and posts it.
type CharacterFlavorExecutor struct {
	llm ports.LLM
}

// NewCharacterFlavorExecutor constructs a character flavor executor.
func NewCharacterFlavorExecutor(llm ports.LLM) *CharacterFlavorExecutor {
	return &CharacterFlavorExecutor{llm: llm}
}

// Execute renders the flavor prompt and publishes the strict JSON result.
func (e *CharacterFlavorExecutor) Execute(ctx context.Context, effect domain.CharacterFlavor, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil || e.llm == nil {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	template, err := prompts.TemplateFor(vocab.RoleCharacterFlavor)
	if err != nil {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	rendered, err := template.Render(map[string]string{
		"species": effect.Species, "gender": effect.Gender, "class": effect.Class,
		"background": effect.Background, "hooks": "the supplied hero's unresolved past",
	})
	if err != nil {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	schema, err := prompts.SchemaFor(vocab.RoleCharacterFlavor)
	if err != nil {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	raw, err := e.llm.JSON(ctx, ports.TextRequest{
		Meta: ports.CallMeta{Role: vocab.RoleCharacterFlavor, Phase: vocab.StateCreation, Seat: effect.Seat},
		Messages: []ports.Message{
			{Role: vocab.MsgSystem, Text: template.System},
			{Role: vocab.MsgUser, Text: rendered[len(template.System)+2:]},
		},
		MaxTokens: 160,
	}, ports.Schema{Name: string(vocab.RoleCharacterFlavor), JSON: schema.JSON})
	if err != nil || schema.Validate(raw) != nil {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	var value struct {
		Name string `json:"name"`
		Look string `json:"look"`
		Hook string `json:"hook"`
	}
	if err := json.Unmarshal(raw, &value); err != nil || value.Name == "" || value.Look == "" || value.Hook == "" {
		post(ctx, in, scope, domain.FlavorFailed{Seat: effect.Seat})
		return
	}
	post(ctx, in, scope, domain.FlavorDone{Seat: effect.Seat, Flavor: domain.Flavor{Name: value.Name, Look: value.Look, Hook: value.Hook}})
}

func post(ctx context.Context, in ports.Inbox, scope domain.Scope, event domain.Event) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: event})
	}
}

func postLineFailure(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID, kind vocab.ErrKind) {
	post(ctx, in, scope, domain.LineFailed{UtteranceID: id, FailureKind: kind})
}

func failureKind(err error) vocab.ErrKind {
	if err == nil {
		return vocab.ErrUnavailable
	}
	var callErr *ports.CallError
	if errors.As(err, &callErr) && callErr.Kind != "" {
		return callErr.Kind
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return vocab.ErrTimeout
	}
	return vocab.ErrUnavailable
}
