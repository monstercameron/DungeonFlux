package in

import (
	"context"
	"errors"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Transcriber executes the Transcribe effect for one completed recording.
type Transcriber struct {
	stt       ports.STT
	assembler *Assembler
	// Locales resolves per-seat locales for the speech-to-text language
	// hint. A nil resolver keeps English. Composition wires it to room state.
	Locales ports.SeatLocales
}

// NewTranscriber constructs a Transcribe executor over an STT adapter and
// recording assembler.
func NewTranscriber(stt ports.STT, assembler *Assembler) (*Transcriber, error) {
	if stt == nil {
		return nil, errors.New("voice/in: STT adapter is required")
	}
	if assembler == nil {
		return nil, errors.New("voice/in: assembler is required")
	}
	return &Transcriber{stt: stt, assembler: assembler}, nil
}

// Execute transcribes one completed utterance and posts its result to inbox.
// The runtime owns the work goroutine and supplies the scope for the event.
func (e *Transcriber) Execute(ctx context.Context, effect domain.Transcribe, scope domain.Scope, inbox ports.Inbox) {
	if e == nil || e.stt == nil || e.assembler == nil {
		postSTTError(ctx, inbox, scope, effect.UtteranceID, vocab.ErrUnavailable)
		return
	}
	recording, ok, err := e.assembler.Take(ctx, effect.UtteranceID)
	if err != nil {
		postSTTError(ctx, inbox, scope, effect.UtteranceID, classifyError(err))
		return
	}
	if !ok || len(recording.Audio) == 0 {
		postSTTError(ctx, inbox, scope, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	transcript, err := e.stt.Transcribe(ctx, ports.STTRequest{
		Meta:     ports.CallMeta{Seat: effect.Seat, UtteranceID: effect.UtteranceID, Locale: e.localeFor(effect.Seat)},
		Audio:    recording.Audio,
		MIME:     recording.MIME,
		Keyterms: append([]string(nil), effect.Keyterms...),
		Language: sttLanguage(e.localeFor(effect.Seat)),
	})
	if err != nil {
		postSTTError(ctx, inbox, scope, effect.UtteranceID, classifyError(err))
		return
	}
	if strings.TrimSpace(transcript.Text) == "" {
		postSTTError(ctx, inbox, scope, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	postEvent(ctx, inbox, scope, domain.Transcribed{UtteranceID: effect.UtteranceID, Text: transcript.Text})
}

func postSTTError(ctx context.Context, inbox ports.Inbox, scope domain.Scope, utteranceID domain.UtteranceID, kind vocab.ErrKind) {
	postEvent(ctx, inbox, scope, domain.STTError{UtteranceID: utteranceID, FailureKind: kind})
}

func postEvent(ctx context.Context, inbox ports.Inbox, scope domain.Scope, event domain.Event) {
	if inbox == nil {
		return
	}
	inbox.Post(ctx, domain.Envelope{Scope: scope, Event: event})
}

func classifyError(err error) vocab.ErrKind {
	var callErr *ports.CallError
	if errors.As(err, &callErr) && callErr.Kind != "" {
		return callErr.Kind
	}
	if errors.Is(err, context.Canceled) {
		return vocab.ErrCanceled
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return vocab.ErrTimeout
	}
	return vocab.ErrUnavailable
}
