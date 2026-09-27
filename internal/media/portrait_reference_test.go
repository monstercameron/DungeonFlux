package media

import (
	"context"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/fakes"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestPortraitExecutor_UsesReferenceAwareImageInput(t *testing.T) {
	images := &recordingReferenceImageGen{FakeImageGen: &fakes.FakeImageGen{
		Script: []fakes.ImageResult{{Events: []ports.ImageEvent{{PNG: []byte("png")}}}},
	}}
	refs := completeReferences()
	writer := &fakes.FakeAssetWriter{Script: []fakes.AssetWriteResult{{Asset: domain.Asset{ID: "portrait"}}}}
	in := &fakes.FakeInbox{PostResult: true}
	NewPortraitExecutor(PortraitConfig{
		Images: images, Assets: writer, References: map[domain.SeatID]ReferenceAssets{1: refs},
		ReferenceSource: func(_ context.Context, id domain.AssetID) ([]byte, error) { return []byte(id), nil },
	}).Execute(context.Background(), domain.GenerateImage{Slot: "portrait:1", Prompt: "hero"}, domain.Scope{}, in)
	if len(images.References) != 5 || !strings.Contains(images.Calls[0].Request.Prompt, "front") {
		t.Fatalf("references=%d prompt=%q", len(images.References), images.Calls[0].Request.Prompt)
	}
}

type recordingReferenceImageGen struct {
	*fakes.FakeImageGen
	References [][]byte
}

func (g *recordingReferenceImageGen) GenerateWithReferences(ctx context.Context, request ports.ImageRequest, references [][]byte) (ports.ImageStream, error) {
	g.References = references
	return g.Generate(ctx, request)
}

func completeReferences() ReferenceAssets {
	return ReferenceAssets{Sheet: "sheet", Angles: map[vocab.ReferenceAngle]domain.AssetID{
		vocab.ReferenceFront: "front", vocab.ReferenceThreeQuarter: "three", vocab.ReferenceSide: "side", vocab.ReferenceBack: "back",
	}}
}
