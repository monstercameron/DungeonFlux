package media

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/budget"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestReferenceExecutor_GeneratesSheetCropsAndSettlesBudget(t *testing.T) {
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{
		{Asset: domain.Asset{ID: "sheet"}}, {Asset: domain.Asset{ID: "front"}},
		{Asset: domain.Asset{ID: "three"}}, {Asset: domain.Asset{ID: "side"}}, {Asset: domain.Asset{ID: "back"}},
	}}
	images := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Events: []ports.ImageEvent{{PNG: turnaroundPNG(t)}}}}}
	ledger, err := budget.NewLedger(map[vocab.VendorName]float64{vocab.VendorOpenAI: 1})
	if err != nil {
		t.Fatal(err)
	}
	in := &fakes.FakeInbox{PostResult: true}
	NewReferenceExecutor(ReferenceConfig{Images: images, Assets: writer, Budget: ledger}).Execute(context.Background(), domain.GenerateCharacterReference{Seat: 1, Species: "elf", Gender: "female", Class: "rogue", Name: "Mira", Flavor: "river debt"}, domain.Scope{}, in)
	if len(writer.Calls) != 5 || len(in.Calls) != 5 {
		t.Fatalf("writes=%d events=%d", len(writer.Calls), len(in.Calls))
	}
	if !strings.Contains(images.Calls[0].Request.Prompt, "Mira") || images.Calls[0].Request.Size != "1536x1024" {
		t.Fatalf("request=%#v", images.Calls[0].Request)
	}
	if got := ledger.Costs().ByVendor[vocab.VendorOpenAI].Spent; got != 0.07 {
		t.Fatalf("spent=%v", got)
	}
	if event := in.Calls[4].Envelope.Event.(domain.AssetReady); event.Slot != "reference:1:back" {
		t.Fatalf("last event=%#v", event)
	}
}

func TestReferenceExecutor_FailureUsesTemplateSheetAndAngles(t *testing.T) {
	fallback := turnaroundPNG(t)
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{
		{Asset: domain.Asset{ID: "sheet"}}, {Asset: domain.Asset{ID: "front"}},
		{Asset: domain.Asset{ID: "three"}}, {Asset: domain.Asset{ID: "side"}}, {Asset: domain.Asset{ID: "back"}},
	}}
	images := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Err: errors.New("vendor down")}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewReferenceExecutor(ReferenceConfig{Images: images, Assets: writer, FallbackSheet: fallback}).Execute(context.Background(), domain.GenerateCharacterReference{Seat: 2}, domain.Scope{}, in)
	if len(writer.Calls) != 5 || len(in.Calls) != 5 {
		t.Fatalf("writes=%d events=%d", len(writer.Calls), len(in.Calls))
	}
	for _, call := range in.Calls {
		if _, ok := call.Envelope.Event.(domain.AssetReady); !ok {
			t.Fatalf("event=%T", call.Envelope.Event)
		}
	}
}

func TestReferenceExecutor_CancelAndInvalidImageStillPublishFailureFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	images := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Events: []ports.ImageEvent{{PNG: []byte("not png")}}}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewReferenceExecutor(ReferenceConfig{Images: images, Assets: &fakes.FakeAssetWriter{}}).Execute(ctx, domain.GenerateCharacterReference{Seat: 1}, domain.Scope{}, in)
	if len(in.Calls) != 5 {
		t.Fatalf("events=%d", len(in.Calls))
	}
	for _, call := range in.Calls {
		if _, ok := call.Envelope.Event.(domain.AssetReady); !ok {
			t.Fatalf("event=%T", call.Envelope.Event)
		}
	}
}

func turnaroundPNG(t *testing.T) []byte {
	t.Helper()
	canvas := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for index, fill := range []color.RGBA{{R: 1, A: 255}, {G: 2, A: 255}, {B: 3, A: 255}, {R: 4, G: 5, A: 255}} {
		for x := index * 2; x < (index+1)*2; x++ {
			for y := 0; y < 4; y++ {
				canvas.Set(x, y, fill)
			}
		}
	}
	var data bytes.Buffer
	if err := png.Encode(&data, canvas); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
