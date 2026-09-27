package creation

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/game/rules"
)

func TestSlots_RequestsPortraitAndFlavorInParallel(t *testing.T) {
	slots, err := NewSlots([2]rules.Class{rules.Paladin, rules.Rogue})
	if err != nil {
		t.Fatal(err)
	}
	seats := []SeatState{
		{Seat: 1, Class: rules.Paladin, Species: "human", Gender: "female", Built: true},
		{Seat: 2, Class: rules.Rogue, Species: "elf", Gender: "male", Built: true},
	}
	requests := slots.Requests(seats)
	if len(requests) != 4 {
		t.Fatalf("request count=%d", len(requests))
	}
	image, ok := requests[1].(domain.GenerateImage)
	if !ok || image.Slot != "portrait:1" || !image.Transparent || image.Partials != 2 {
		t.Fatalf("portrait request=%#v", requests[1])
	}
}

func TestSlots_PortraitDeadlineUsesClassFallback(t *testing.T) {
	slots, err := NewSlots([2]rules.Class{rules.Paladin, rules.Rogue})
	if err != nil {
		t.Fatal(err)
	}
	if err := slots.Step(domain.TimerFired{Name: "seat_deadline:1"}); err != nil {
		t.Fatal(err)
	}
	seat, _ := slots.Seat(1)
	asset, ok := seat.Portrait.Asset()
	if !ok || asset.ID != "portrait_fallback_paladin" || seat.Portrait.State() != "fallback" {
		t.Fatalf("fallback seat=%#v asset=%#v ok=%v", seat, asset, ok)
	}
	if err := slots.Step(domain.AssetReady{Slot: "portrait:1", Asset: domain.Asset{ID: "late"}}); err != nil {
		t.Fatal(err)
	}
	seat, _ = slots.Seat(1)
	asset, _ = seat.Portrait.Asset()
	if asset.ID != "portrait_fallback_paladin" {
		t.Fatalf("late result replaced fallback: %#v", asset)
	}
}

func TestSlots_FailedPortraitFallsBackAndFlavorDefault(t *testing.T) {
	slots, err := NewSlots([2]rules.Class{rules.Paladin, rules.Rogue})
	if err != nil {
		t.Fatal(err)
	}
	if err := slots.Step(domain.AssetFailed{Slot: "portrait:2"}); err != nil {
		t.Fatal(err)
	}
	if err := slots.Step(domain.FlavorFailed{Seat: 2}); err != nil {
		t.Fatal(err)
	}
	seat, _ := slots.Seat(2)
	asset, ok := seat.Portrait.Asset()
	if !ok || asset.ID != "portrait_fallback_rogue" {
		t.Fatalf("failed portrait=%#v ok=%v", asset, ok)
	}
	if seat.State != FlavorFallback || seat.Flavor.Name != "Rogue" {
		t.Fatalf("flavor=%#v state=%q", seat.Flavor, seat.State)
	}
}

func TestSlots_ReadyPortraitAndFlavor(t *testing.T) {
	slots, err := NewSlots([2]rules.Class{rules.Paladin, rules.Rogue})
	if err != nil {
		t.Fatal(err)
	}
	if err := slots.Step(domain.AssetReady{Slot: "portrait:2", Asset: domain.Asset{ID: "portrait-2"}}); err != nil {
		t.Fatal(err)
	}
	if err := slots.Step(domain.FlavorDone{Seat: 2, Flavor: domain.Flavor{Name: "Mira"}}); err != nil {
		t.Fatal(err)
	}
	seat, _ := slots.Seat(2)
	asset, ok := seat.Portrait.Asset()
	if !ok || asset.ID != "portrait-2" || seat.State != FlavorReady || seat.Flavor.Name != "Mira" {
		t.Fatalf("seat=%#v asset=%#v ok=%v", seat, asset, ok)
	}
}
