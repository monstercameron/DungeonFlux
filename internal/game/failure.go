package game

import (
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// FailureEvent returns the deterministic event to post when a work effect has
// no registered executor. Control effects do not have a failure event because
// they are handled by the runtime itself.
func FailureEvent(effect domain.Effect) (domain.Event, bool) {
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
	case domain.PrerenderText:
		return domain.PrerenderFailed{Set: value.Set}, true
	case domain.RenderLines:
		return domain.PrerenderFailed{Set: value.Set}, true
	case domain.GenerateImage:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.ComposeStill:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.GenerateClip:
		return domain.AssetFailed{Slot: value.Slot, FailureKind: vocab.ErrUnavailable}, true
	case domain.GenerateBillboardLoops:
		return domain.AssetFailed{FailureKind: vocab.ErrUnavailable}, true
	default:
		return nil, false
	}
}
