package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestComposeStillExecutor_CompositesTransparentLayer(t *testing.T) {
	background := solidPNG(t, color.RGBA{R: 10, A: 255})
	layer := solidPNG(t, color.RGBA{B: 200, A: 128})
	assets := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "still"}}}}
	source := func(_ context.Context, id domain.AssetID) ([]byte, error) {
		if id == "background" {
			return background, nil
		}
		return layer, nil
	}
	in := &fakes.FakeInbox{PostResult: true}
	NewComposeStillExecutor(ComposeStillConfig{Source: source, Assets: assets}).Execute(context.Background(), domain.ComposeStill{Slot: "scene", Background: "background", Layers: []string{"hero"}}, domain.Scope{}, in)
	if len(assets.Calls) != 1 {
		t.Fatalf("writes = %d", len(assets.Calls))
	}
	decoded, err := png.Decode(bytes.NewReader(assets.Calls[0].Data))
	if err != nil {
		t.Fatal(err)
	}
	got := decoded.At(0, 0)
	if got == (color.RGBA{R: 10, A: 255}) {
		t.Fatal("layer was not composited")
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
}

func TestComposeStillExecutor_PassesReadyReferencesToConditioningHook(t *testing.T) {
	background := solidPNG(t, color.RGBA{R: 10, A: 255})
	assets := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "still"}}}}
	var got []domain.AssetID
	refs := ReferenceAssets{Sheet: "sheet", Angles: map[vocab.ReferenceAngle]domain.AssetID{
		vocab.ReferenceFront: "front", vocab.ReferenceThreeQuarter: "three", vocab.ReferenceSide: "side", vocab.ReferenceBack: "back",
	}}
	in := &fakes.FakeInbox{PostResult: true}
	NewComposeStillExecutor(ComposeStillConfig{
		Source: func(context.Context, domain.AssetID) ([]byte, error) { return background, nil }, Assets: assets,
		References: map[domain.SeatID]ReferenceAssets{1: refs}, ReferenceInput: func(_ context.Context, ids []domain.AssetID) error { got = ids; return nil },
	}).Execute(context.Background(), domain.ComposeStill{Slot: "scene:1", Background: "background"}, domain.Scope{}, in)
	if len(got) != 5 || got[0] != "sheet" || got[4] != "back" {
		t.Fatalf("reference IDs=%v", got)
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event=%T", in.Calls[0].Envelope.Event)
	}
}

func TestComposeStillExecutor_InvalidInputFails(t *testing.T) {
	assets := &fakes.FakeAssetWriter{}
	in := &fakes.FakeInbox{PostResult: true}
	e := NewComposeStillExecutor(ComposeStillConfig{Source: func(context.Context, domain.AssetID) ([]byte, error) { return nil, errors.New("missing") }, Assets: assets})
	e.Execute(context.Background(), domain.ComposeStill{Slot: "scene", Background: "missing"}, domain.Scope{}, in)
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
	if len(assets.Calls) != 0 {
		t.Fatal("invalid source was written")
	}
}

func TestComposeStillExecutor_RejectsMissingBackground(t *testing.T) {
	in := &fakes.FakeInbox{PostResult: true}
	NewComposeStillExecutor(ComposeStillConfig{Source: func(context.Context, domain.AssetID) ([]byte, error) { return nil, nil }, Assets: &fakes.FakeAssetWriter{}}).Execute(context.Background(), domain.ComposeStill{Slot: "scene"}, domain.Scope{}, in)
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
}

func solidPNG(t *testing.T, fill color.Color) []byte {
	t.Helper()
	imageValue := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			imageValue.Set(x, y, fill)
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, imageValue); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
