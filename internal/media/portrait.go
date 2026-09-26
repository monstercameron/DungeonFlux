package media

import (
	"context"
	"errors"
	"io"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// PortraitConfig supplies the dependencies for a portrait executor.
type PortraitConfig struct {
	Images         ports.ImageGen
	Assets         ports.AssetWriter
	Pool           *Pool
	Fallback       []byte
	Fallbacks      map[string][]byte
	FallbackVendor vocab.VendorName
}

// PortraitExecutor runs GenerateImage effects and posts asset events.
type PortraitExecutor struct {
	images    ports.ImageGen
	assets    ports.AssetWriter
	pool      *Pool
	fallback  []byte
	fallbacks map[string][]byte
}

// NewPortraitExecutor constructs a portrait executor from injected services.
func NewPortraitExecutor(config PortraitConfig) *PortraitExecutor {
	return &PortraitExecutor{
		images: config.Images, assets: config.Assets, pool: config.Pool,
		fallback: append([]byte(nil), config.Fallback...), fallbacks: cloneBytes(config.Fallbacks),
	}
}

// Execute generates a transparent portrait, using a template image when the
// vendor fails. The inbox receives a partial event for each preview and one
// ready or failed terminal event.
func (e *PortraitExecutor) Execute(ctx context.Context, effect domain.GenerateImage, scope domain.Scope, in ports.Inbox) {
	if e == nil || e.images == nil || e.assets == nil {
		e.postFailure(ctx, effect.Slot, vocab.ErrUnavailable, scope, in)
		return
	}
	req := ports.ImageRequest{Prompt: effect.Prompt, Size: effect.Size, Transparent: effect.Transparent, Partials: effect.Partials}
	var stream ports.ImageStream
	var err error
	job := func(run context.Context) error {
		stream, err = e.images.Generate(run, req)
		return err
	}
	if e.pool != nil {
		err = e.pool.Run(ctx, vocab.VendorOpenAI, job)
	} else {
		err = job(ctx)
	}
	if err == nil {
		err = e.consume(ctx, effect.Slot, stream, scope, in)
	}
	if err == nil {
		return
	}
	if stream != nil {
		_ = stream.Close()
	}
	fallback := e.fallbackFor(effect.Slot)
	if len(fallback) == 0 {
		e.postFailure(ctx, effect.Slot, failureKind(err), scope, in)
		return
	}
	asset, writeErr := e.assets.Write(ctx, vocab.AssetImage, "image/png", fallback, ports.AssetMeta{})
	if writeErr != nil {
		e.postFailure(ctx, effect.Slot, failureKind(writeErr), scope, in)
		return
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: effect.Slot, Asset: asset}})
}

func (e *PortraitExecutor) consume(ctx context.Context, slot string, stream ports.ImageStream, scope domain.Scope, in ports.Inbox) error {
	if stream == nil {
		return errors.New("portrait: image stream is nil")
	}
	defer stream.Close()
	var ready bool
	var last domain.Asset
	var lastPartial bool
	for {
		event, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if !ready {
				return errors.New("portrait: stream had no image")
			}
			if lastPartial {
				post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: slot, Asset: last}})
			}
			return nil
		}
		if err != nil {
			return err
		}
		if len(event.PNG) == 0 {
			continue
		}
		asset, err := e.assets.Write(ctx, vocab.AssetImage, "image/png", event.PNG, ports.AssetMeta{})
		if err != nil {
			return err
		}
		ready = true
		last, lastPartial = asset, event.Partial
		if event.Partial {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetPartial{Slot: slot, Asset: asset}})
		} else {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: slot, Asset: asset}})
		}
	}
}

func (e *PortraitExecutor) fallbackFor(slot string) []byte {
	if value, ok := e.fallbacks[slot]; ok {
		return append([]byte(nil), value...)
	}
	return append([]byte(nil), e.fallback...)
}

func (e *PortraitExecutor) postFailure(ctx context.Context, slot string, kind vocab.ErrKind, scope domain.Scope, in ports.Inbox) {
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: slot, FailureKind: kind}})
}

func failureKind(err error) vocab.ErrKind {
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

func cloneBytes(values map[string][]byte) map[string][]byte {
	if len(values) == 0 {
		return nil
	}
	copyValues := make(map[string][]byte, len(values))
	for key, value := range values {
		copyValues[key] = append([]byte(nil), value...)
	}
	return copyValues
}

func post(ctx context.Context, in ports.Inbox, envelope domain.Envelope) {
	if in == nil || !in.Post(ctx, envelope) {
		return
	}
}
