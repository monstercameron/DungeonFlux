package media

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// StillSource loads PNG bytes for a logical asset ID.
type StillSource func(context.Context, domain.AssetID) ([]byte, error)

// ComposeStillConfig supplies the asset source and destination writer.
type ComposeStillConfig struct {
	Source         StillSource
	Assets         ports.AssetWriter
	References     map[domain.SeatID]ReferenceAssets
	ReferenceInput func(context.Context, []domain.AssetID) error
}

// ComposeStillExecutor composites transparent layers over a background.
type ComposeStillExecutor struct {
	source         StillSource
	assets         ports.AssetWriter
	references     map[domain.SeatID]ReferenceAssets
	referenceInput func(context.Context, []domain.AssetID) error
}

// NewComposeStillExecutor constructs a still compositor.
func NewComposeStillExecutor(config ComposeStillConfig) *ComposeStillExecutor {
	return &ComposeStillExecutor{source: config.Source, assets: config.Assets, references: cloneReferenceAssets(config.References), referenceInput: config.ReferenceInput}
}

// Execute runs a ComposeStill effect and posts a ready or failed asset event.
func (e *ComposeStillExecutor) Execute(ctx context.Context, effect domain.ComposeStill, scope domain.Scope, in ports.Inbox) {
	data, err := e.compose(ctx, effect)
	if err == nil {
		var asset domain.Asset
		asset, err = e.assets.Write(ctx, vocab.AssetImage, "image/png", data, ports.AssetMeta{})
		if err == nil {
			post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetReady{Slot: effect.Slot, Asset: asset}})
			return
		}
	}
	post(ctx, in, domain.Envelope{Scope: scope, Event: domain.AssetFailed{Slot: effect.Slot, FailureKind: failureKind(err)}})
}

func (e *ComposeStillExecutor) compose(ctx context.Context, effect domain.ComposeStill) ([]byte, error) {
	if e == nil || e.source == nil || e.assets == nil {
		return nil, fmt.Errorf("compose still: dependencies are incomplete")
	}
	if effect.Background == "" {
		return nil, fmt.Errorf("compose still: background is required")
	}
	if references := e.references[seatFromMediaSlot(effect.Slot)]; references.Ready() && e.referenceInput != nil {
		if err := e.referenceInput(ctx, references.IDs()); err != nil {
			return nil, fmt.Errorf("load reference inputs: %w", err)
		}
	}
	background, err := e.loadPNG(ctx, effect.Background)
	if err != nil {
		return nil, fmt.Errorf("load background: %w", err)
	}
	canvas := image.NewRGBA(background.Bounds())
	draw.Draw(canvas, canvas.Bounds(), background, background.Bounds().Min, draw.Src)
	for _, id := range effect.Layers {
		layer, loadErr := e.loadPNG(ctx, domain.AssetID(id))
		if loadErr != nil {
			return nil, fmt.Errorf("load layer %q: %w", id, loadErr)
		}
		draw.Draw(canvas, canvas.Bounds(), layer, layer.Bounds().Min, draw.Over)
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, fmt.Errorf("encode still: %w", err)
	}
	return output.Bytes(), nil
}

func (e *ComposeStillExecutor) loadPNG(ctx context.Context, id domain.AssetID) (image.Image, error) {
	data, err := e.source(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("asset %q is empty", id)
	}
	value, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("asset %q is not PNG: %w", id, err)
	}
	return value, nil
}
