package runtime

import (
	"context"
	"io"
	"log/slog"
	"reflect"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// Executor runs one work effect and posts its result through in.
type Executor[E domain.Effect] func(context.Context, E, domain.Scope, ports.Inbox)

type effectExecutor func(context.Context, domain.Effect, domain.Scope, ports.Inbox)

// Runner dispatches engine effects to registered executors.
type Runner struct {
	in       ports.Inbox
	logger   *slog.Logger
	registry map[reflect.Type]effectExecutor
}

// NewRunner constructs an empty executor registry.
func NewRunner(in ports.Inbox, logger *slog.Logger) *Runner {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Runner{in: in, logger: logger, registry: make(map[reflect.Type]effectExecutor)}
}

// Handle registers fn for effects of type E. Registering a type twice is a
// programmer error and panics during startup.
func Handle[E domain.Effect](r *Runner, fn Executor[E]) {
	typ := reflect.TypeOf((*E)(nil)).Elem()
	if _, exists := r.registry[typ]; exists {
		panic("runtime: duplicate effect executor registration for " + typ.String())
	}
	r.registry[typ] = func(ctx context.Context, effect domain.Effect, scope domain.Scope, in ports.Inbox) {
		fn(ctx, effect.(E), scope, in)
	}
}

// Run dispatches effects in execution order. Work effects receive independent
// goroutines; control effects are reserved for the control runners.
func (r *Runner) Run(effects []domain.Effect) {
	for _, effect := range effects {
		r.dispatch(effect)
	}
}

func (r *Runner) dispatch(effect domain.Effect) {
	if isControl(effect) {
		return
	}
	typ := reflect.TypeOf(effect)
	fn, ok := r.registry[typ]
	if !ok {
		r.unregistered(effect)
		return
	}
	go runRecovered(context.Background(), r.logger, effect, domain.Scope{}, r.in, fn)
}

func isControl(effect domain.Effect) bool {
	switch effect.(type) {
	case domain.StartTimer, domain.CancelTimer, domain.FreezeTimer, domain.ThawTimer,
		domain.PauseAll, domain.ResumeAll, domain.CancelScope, domain.CancelKey, domain.NewRun,
		domain.TalkStop, domain.SendAudioCancel, domain.Checkpoint:
		return true
	default:
		return false
	}
}

func (r *Runner) unregistered(effect domain.Effect) {
	r.logger.Warn("unregistered effect", "effect", effect.Kind())
	if r.in == nil {
		return
	}
	if event, ok := failureEvent(effect); ok {
		r.in.Post(context.Background(), domain.Envelope{Scope: effectScope(effect), Event: event})
	}
}

func effectScope(effect domain.Effect) domain.Scope {
	if timer, ok := effect.(domain.TimerEffect); ok {
		return timer.Scope
	}
	if timer, ok := effect.(domain.StartTimer); ok {
		return domain.TimerEffect(timer).Scope
	}
	return domain.Scope{}
}

func failureEvent(effect domain.Effect) (domain.Event, bool) {
	switch value := effect.(type) {
	case domain.Transcribe:
		return domain.STTError{UtteranceID: value.UtteranceID, FailureKind: vocab.ErrUnavailable}, true
	case domain.Interpret:
		return domain.InterpretFailed{UtteranceID: value.UtteranceID}, true
	case domain.CharacterFlavor:
		return domain.FlavorFailed{Seat: value.Seat}, true
	case domain.StartLine:
		return domain.LineFailed{UtteranceID: value.UtteranceID, FailureKind: vocab.ErrUnavailable}, true
	case domain.PlayCanned:
		return domain.LineFailed{UtteranceID: value.UtteranceID, FailureKind: vocab.ErrUnavailable}, true
	case domain.GenerateImage:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.ComposeStill:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.GenerateClip:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.GenerateBillboardLoops:
		return domain.AssetFailed{FailureKind: vocab.ErrUnavailable}, true
	case domain.PrerenderText:
		return domain.PrerenderFailed{Set: value.Set}, true
	case domain.RenderLines:
		return domain.PrerenderFailed{Set: value.Set}, true
	default:
		return nil, false
	}
}
