package media

import (
	"context"
	"errors"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPortraitExecutor_ReadyAfterPartial(t *testing.T) {
	asset := domain.Asset{ID: "portrait"}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: asset}, {Asset: asset}}}
	image := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Events: []ports.ImageEvent{{PNG: []byte("p"), Partial: true}, {PNG: []byte("f")}}}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewPortraitExecutor(PortraitConfig{Images: image, Assets: writer}).Execute(context.Background(), domain.GenerateImage{Slot: "hero", Prompt: "hero", Transparent: true}, domain.Scope{}, in)
	if len(in.Calls) != 2 {
		t.Fatalf("events = %d, want partial and ready", len(in.Calls))
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetPartial); !ok {
		t.Fatalf("first event = %T", in.Calls[0].Envelope.Event)
	}
	if _, ok := in.Calls[1].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("second event = %T", in.Calls[1].Envelope.Event)
	}
}

func TestPortraitExecutor_UsesSlotFallbackOnVendorError(t *testing.T) {
	asset := domain.Asset{ID: "fallback"}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: asset}}}
	image := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Err: errors.New("vendor down")}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewPortraitExecutor(PortraitConfig{Images: image, Assets: writer, Fallbacks: map[string][]byte{"hero": []byte("fallback")}}).Execute(context.Background(), domain.GenerateImage{Slot: "hero", Prompt: "hero"}, domain.Scope{}, in)
	if len(writer.Calls) != 1 || string(writer.Calls[0].Data) != "fallback" {
		t.Fatalf("fallback write = %+v", writer.Calls)
	}
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
}

func TestPortraitExecutor_FailsWithoutFallback(t *testing.T) {
	image := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Err: errors.New("vendor down")}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewPortraitExecutor(PortraitConfig{Images: image, Assets: &fakes.FakeAssetWriter{}}).Execute(context.Background(), domain.GenerateImage{Slot: "hero"}, domain.Scope{}, in)
	event, ok := in.Calls[0].Envelope.Event.(domain.AssetFailed)
	if !ok || event.FailureKind != vocab.ErrUnavailable {
		t.Fatalf("event = %#v", in.Calls[0].Envelope.Event)
	}
}

func TestPortraitExecutor_EmptyStreamFallsBack(t *testing.T) {
	image := &fakes.FakeImageGen{Script: []fakes.ImageResult{{Events: nil}}}
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "fallback"}}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewPortraitExecutor(PortraitConfig{Images: image, Assets: writer, Fallback: []byte("fallback")}).Execute(context.Background(), domain.GenerateImage{Slot: "hero"}, domain.Scope{}, in)
	if _, ok := in.Calls[0].Envelope.Event.(domain.AssetReady); !ok {
		t.Fatalf("event = %T", in.Calls[0].Envelope.Event)
	}
}

func TestFailureKind_ContextAndCallErrors(t *testing.T) {
	if got := failureKind(context.DeadlineExceeded); got != vocab.ErrTimeout {
		t.Fatal(got)
	}
	if got := failureKind(context.Canceled); got != vocab.ErrCanceled {
		t.Fatal(got)
	}
	if got := failureKind(&ports.CallError{Kind: vocab.ErrAuth}); got != vocab.ErrAuth {
		t.Fatal(got)
	}
}
