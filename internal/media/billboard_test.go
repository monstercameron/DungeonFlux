package media

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestBillboardReferenceFrames_UsesSideThenFront(t *testing.T) {
	refs := ReferenceAssets{Sheet: "sheet", Angles: map[vocab.ReferenceAngle]domain.AssetID{
		vocab.ReferenceSide: "side", vocab.ReferenceFront: "front", vocab.ReferenceThreeQuarter: "three", vocab.ReferenceBack: "back",
	}}
	got := BillboardReferenceFrames(refs)
	if len(got) != 2 || got[0] != "side" || got[1] != "front" {
		t.Fatalf("frames=%v", got)
	}
	if BillboardReferenceFrames(ReferenceAssets{}) != nil {
		t.Fatal("incomplete reference should not condition billboard")
	}
}
