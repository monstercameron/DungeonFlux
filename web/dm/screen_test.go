package dm

import (
	"reflect"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestSelectLayers_AllPhases(t *testing.T) {
	tests := []struct {
		phase string
		want  []Layer
	}{
		{"lobby", []Layer{LayerLobby, LayerMusic}},
		{"creation", []Layer{LayerCreation, LayerMusic}},
		// No Clip data on this bare state: the establishing clip layer only
		// mounts once there is a clip to show (see hasClip), otherwise its
		// opaque fallback still hid the scene's title, party and DM box.
		{"opening", []Layer{LayerScene, LayerMusic}},
		{"exploration", []Layer{LayerScene, LayerMusic}},
		{"conversation", []Layer{LayerScene, LayerMusic}},
		{"check", []Layer{LayerScene, LayerMusic, LayerDice}},
		// Resolution narrates the outcome through the scene caption; the dice
		// layer belongs to check, not to the reveal that follows it.
		{"resolution", []Layer{LayerScene, LayerMusic}},
		{"hook_event", []Layer{LayerScene, LayerClip, LayerCallout, LayerMusic}},
		{"combat", []Layer{LayerCombat, LayerDice, LayerMusic}},
		{"cliffhanger", []Layer{LayerScene, LayerClip, LayerMusic}},
		{"end", []Layer{LayerEnd, LayerMusic}},
	}
	for _, test := range tests {
		t.Run(test.phase, func(t *testing.T) {
			got := SelectLayers(&dungeonfluxv1.ScreenState{Phase: test.phase})
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("SelectLayers(%q) = %#v, want %#v", test.phase, got, test.want)
			}
		})
	}
}

func TestSelectLayers_OpeningWithClipDataMountsClipLayer(t *testing.T) {
	state := &dungeonfluxv1.ScreenState{Phase: "opening", View: &dungeonfluxv1.ScreenState_Dm{Dm: &dungeonfluxv1.DMView{
		Clip: &dungeonfluxv1.Clip{Url: "establishing_tavern"},
	}}}
	want := []Layer{LayerScene, LayerClip, LayerMusic}
	if got := SelectLayers(state); !reflect.DeepEqual(got, want) {
		t.Fatalf("SelectLayers(opening with clip) = %#v, want %#v", got, want)
	}
}

func TestSelectLayers_NilAndUnknownUseLobby(t *testing.T) {
	for _, state := range []*dungeonfluxv1.ScreenState{nil, {Phase: "future"}} {
		if got := SelectLayers(state); !reflect.DeepEqual(got, []Layer{LayerLobby}) {
			t.Fatalf("fallback = %#v", got)
		}
	}
}

func TestHasLayer(t *testing.T) {
	layers := []Layer{LayerScene, LayerMusic}
	if !HasLayer(layers, LayerScene) || HasLayer(layers, LayerCombat) {
		t.Fatalf("HasLayer returned an incorrect result")
	}
}

func TestPhaseName_NormalizesEmptyAndUnderscore(t *testing.T) {
	cases := []struct {
		name  string
		state *dungeonfluxv1.ScreenState
		want  string
	}{
		{name: "nil", want: "lobby"},
		{name: "empty", state: &dungeonfluxv1.ScreenState{}, want: "lobby"},
		{name: "hook event", state: &dungeonfluxv1.ScreenState{Phase: " Hook_Event "}, want: "hook-event"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PhaseName(tc.state); got != tc.want {
				t.Fatalf("PhaseName() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestFrameFromState_PreservesLayerOrder(t *testing.T) {
	frame := FrameFromState(&dungeonfluxv1.ScreenState{Phase: "combat"})
	if frame.Phase != "combat" || !HasLayer(frame.Layers, LayerCombat) || HasLayer(frame.Layers, LayerTimer) {
		t.Fatalf("frame = %#v", frame)
	}
}
