package nested

import (
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func asset(id string) domain.Asset { return domain.Asset{ID: domain.AssetID(id), URL: "/assets/" + id} }

func TestMachine_StepLifecycle(t *testing.T) {
	cases := []struct {
		name       string
		events     []domain.Event
		wantState  SlotState
		wantAsset  string
		wantExists bool
	}{
		{name: "ready primary", events: []domain.Event{domain.AssetReady{Asset: asset("primary")}}, wantState: SlotReady, wantAsset: "primary", wantExists: true},
		{name: "failed without fallback", events: []domain.Event{domain.AssetFailed{}}, wantState: SlotFailed},
		{name: "failed uses first fallback", events: []domain.Event{domain.AssetFailed{}}, wantState: SlotFallback, wantAsset: "fallback-a", wantExists: true},
		{name: "deadline uses first fallback", events: []domain.Event{domain.TimerFired{Name: "deadline"}}, wantState: SlotFallback, wantAsset: "fallback-a", wantExists: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fallbacks := []domain.Asset(nil)
			if tc.wantAsset == "fallback-a" {
				fallbacks = []domain.Asset{asset("fallback-a")}
			}
			machine, err := NewSlot("portrait", fallbacks)
			if err != nil {
				t.Fatal(err)
			}
			for _, event := range tc.events {
				if err := machine.Step(event); err != nil {
					t.Fatal(err)
				}
			}
			if machine.State() != tc.wantState {
				t.Fatalf("state = %s, want %s", machine.State(), tc.wantState)
			}
			got, ok := machine.Asset()
			if ok != tc.wantExists || (ok && string(got.ID) != tc.wantAsset) {
				t.Fatalf("asset = %#v, %v", got, ok)
			}
		})
	}
}

func TestMachine_FallbackChainAdvancesAndIgnoresLatePrimary(t *testing.T) {
	machine, err := NewSlot("clip", []domain.Asset{asset("one"), asset("two")})
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Step(domain.AssetFailed{}); err != nil {
		t.Fatal(err)
	}
	if err := machine.Step(domain.AssetFailed{}); err != nil {
		t.Fatal(err)
	}
	got, ok := machine.Asset()
	if !ok || got.ID != domain.AssetID("two") || machine.State() != SlotFallback {
		t.Fatalf("fallback chain = %#v, state %s, ok %v", got, machine.State(), ok)
	}
	if err := machine.Step(domain.AssetReady{Asset: asset("late-primary")}); err != nil {
		t.Fatal(err)
	}
	got, _ = machine.Asset()
	if got.ID != domain.AssetID("two") {
		t.Fatalf("late result replaced fallback with %q", got.ID)
	}
}

func TestMachine_PartialAndView(t *testing.T) {
	machine, err := NewSlot("portrait", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Step(domain.AssetPartial{Asset: asset("preview")}); err != nil {
		t.Fatal(err)
	}
	if machine.State() != SlotPending {
		t.Fatalf("partial changed state to %s", machine.State())
	}
	view := machine.View()
	if view.Name != "portrait" || view.State != string(SlotPending) || view.Asset != "" {
		t.Fatalf("unexpected view: %#v", view)
	}
	if err := machine.Step(domain.AssetReady{Asset: asset("final")}); err != nil {
		t.Fatal(err)
	}
	if machine.View().Asset != domain.AssetID("final") {
		t.Fatal("ready asset missing from view")
	}
}

func TestMachine_RejectsInvalidAndIgnoresUnknownEvents(t *testing.T) {
	if _, err := NewSlot("", nil); err == nil {
		t.Fatal("empty name accepted")
	}
	machine, err := NewSlot("portrait", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := machine.Step(domain.Act{Move: vocab.MoveReady}); err == nil {
		t.Fatal("non-slot event accepted")
	}
	if err := machine.Step(nil); err == nil {
		t.Fatal("nil event accepted")
	}
	if err := machine.Step(domain.AssetReady{Asset: asset("ready")}); err != nil {
		t.Fatal(err)
	}
	if err := machine.Step(domain.AssetReady{Slot: "other", Asset: asset("other")}); err != nil {
		t.Fatal(err)
	}
	if got, _ := machine.Asset(); got.ID != domain.AssetID("ready") {
		t.Fatalf("other slot changed asset to %q", got.ID)
	}
	if err := machine.Step(domain.AssetFailed{}); err != nil {
		t.Fatal(err)
	}
	if machine.State() != SlotReady {
		t.Fatal("late failure changed ready state")
	}
}
