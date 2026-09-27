package media

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// BillboardReferenceFrames returns the side and front crops in the order used
// by combat billboard generation. Empty or partial references return no IDs.
func BillboardReferenceFrames(references ReferenceAssets) []domain.AssetID {
	if !references.Ready() {
		return nil
	}
	return []domain.AssetID{references.Angles[vocab.ReferenceSide], references.Angles[vocab.ReferenceFront]}
}
