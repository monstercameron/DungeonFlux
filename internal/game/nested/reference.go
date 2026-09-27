package nested

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ReferenceState is the lifecycle of a character reference sheet.
type ReferenceState string

const (
	// ReferencePending means one or more angle assets are still outstanding.
	ReferencePending ReferenceState = "pending"
	// ReferenceReady means the sheet and all four angle assets are available.
	ReferenceReady ReferenceState = "ready"
	// ReferenceFailed means generation failed and callers should use templates.
	ReferenceFailed ReferenceState = "failed"
)

// ReferenceSlot tracks the sheet and per-angle assets for one character.
type ReferenceSlot struct {
	Seat   domain.SeatID
	State  ReferenceState
	Sheet  domain.AssetID
	Assets map[vocab.ReferenceAngle]domain.AssetID
}

// NewReferenceSlot creates a pending reference slot for a seat.
func NewReferenceSlot(seat domain.SeatID) (ReferenceSlot, error) {
	if seat < 1 {
		return ReferenceSlot{}, errors.New("reference seat must be positive")
	}
	return ReferenceSlot{Seat: seat, State: ReferencePending, Assets: make(map[vocab.ReferenceAngle]domain.AssetID)}, nil
}

// Step applies one sheet or angle asset callback. Late callbacks after a
// failure or ready state are ignored, matching the ordinary asset slot.
func (s *ReferenceSlot) Step(event domain.Event) error {
	if s == nil || event == nil {
		return errors.New("reference slot or event is nil")
	}
	if s.State == ReferenceReady || s.State == ReferenceFailed {
		return nil
	}
	slot, asset, failed, ok := referenceEvent(event)
	if !ok {
		return fmt.Errorf("event %q is not a reference asset event", event.Kind())
	}
	seat, angle, sheet, ok := parseReferenceSlot(slot)
	if !ok || seat != s.Seat {
		return nil
	}
	if failed {
		s.State = ReferenceFailed
		return nil
	}
	if sheet {
		s.Sheet = asset.ID
	} else {
		s.Assets[angle] = asset.ID
	}
	if s.Sheet != "" && len(s.Assets) == len(vocab.ReferenceAngles()) {
		s.State = ReferenceReady
	}
	return nil
}

// AssetIDs returns a copy of the ready angle IDs in canonical order.
func (s ReferenceSlot) AssetIDs() map[vocab.ReferenceAngle]domain.AssetID {
	result := make(map[vocab.ReferenceAngle]domain.AssetID, len(s.Assets))
	for angle, id := range s.Assets {
		result[angle] = id
	}
	return result
}

func referenceEvent(event domain.Event) (string, domain.Asset, bool, bool) {
	switch value := event.(type) {
	case domain.AssetReady:
		return value.Slot, value.Asset, false, true
	case domain.AssetFailed:
		return value.Slot, domain.Asset{}, true, true
	default:
		return "", domain.Asset{}, false, false
	}
}

func parseReferenceSlot(slot string) (domain.SeatID, vocab.ReferenceAngle, bool, bool) {
	parts := strings.Split(slot, ":")
	if len(parts) != 3 || parts[0] != "reference" {
		return 0, "", false, false
	}
	seat, err := strconv.Atoi(parts[1])
	if err != nil || seat < 1 {
		return 0, "", false, false
	}
	if parts[2] == "sheet" {
		return domain.SeatID(seat), "", true, true
	}
	for _, angle := range vocab.ReferenceAngles() {
		if parts[2] == string(angle) {
			return domain.SeatID(seat), angle, false, true
		}
	}
	return 0, "", false, false
}
