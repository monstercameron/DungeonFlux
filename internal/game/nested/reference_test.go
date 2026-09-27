package nested

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func TestReferenceSlot_TracksSheetAndAngles(t *testing.T) {
	slot, err := NewReferenceSlot(1)
	if err != nil {
		t.Fatal(err)
	}
	events := []domain.Event{
		domain.AssetReady{Slot: "reference:1:sheet", Asset: domain.Asset{ID: "sheet"}},
		domain.AssetReady{Slot: "reference:1:front", Asset: domain.Asset{ID: "front"}},
		domain.AssetReady{Slot: "reference:1:three_quarter", Asset: domain.Asset{ID: "three"}},
		domain.AssetReady{Slot: "reference:1:side", Asset: domain.Asset{ID: "side"}},
		domain.AssetReady{Slot: "reference:1:back", Asset: domain.Asset{ID: "back"}},
	}
	for _, event := range events {
		if err := slot.Step(event); err != nil {
			t.Fatal(err)
		}
	}
	if slot.State != ReferenceReady || slot.Sheet != "sheet" {
		t.Fatalf("slot=%#v", slot)
	}
	if got := slot.AssetIDs()[vocab.ReferenceSide]; got != "side" {
		t.Fatalf("side=%q", got)
	}
}

func TestReferenceSlot_FailureAndLateCallbacks(t *testing.T) {
	slot, err := NewReferenceSlot(2)
	if err != nil {
		t.Fatal(err)
	}
	if err := slot.Step(domain.AssetFailed{Slot: "reference:2:side"}); err != nil {
		t.Fatal(err)
	}
	if slot.State != ReferenceFailed {
		t.Fatalf("state=%s", slot.State)
	}
	if err := slot.Step(domain.AssetReady{Slot: "reference:2:front", Asset: domain.Asset{ID: "late"}}); err != nil {
		t.Fatal(err)
	}
	if len(slot.AssetIDs()) != 0 {
		t.Fatalf("late asset accepted: %#v", slot.AssetIDs())
	}
}

func TestReferenceSlot_IgnoresOtherSeatAndRejectsUnknown(t *testing.T) {
	slot, err := NewReferenceSlot(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := slot.Step(domain.AssetReady{Slot: "reference:2:front", Asset: domain.Asset{ID: "other"}}); err != nil {
		t.Fatal(err)
	}
	if len(slot.AssetIDs()) != 0 {
		t.Fatal("other seat changed the slot")
	}
	if err := slot.Step(domain.Act{}); err == nil {
		t.Fatal("non-asset event accepted")
	}
}
