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
		{"creation", []Layer{LayerScene, LayerMusic}},
		{"opening", []Layer{LayerScene, LayerClip, LayerMusic}},
		{"exploration", []Layer{LayerScene, LayerMusic}},
		{"conversation", []Layer{LayerScene, LayerMusic}},
		{"check", []Layer{LayerScene, LayerMusic, LayerDice}},
		{"resolution", []Layer{LayerScene, LayerMusic, LayerDice}},
		{"hook_event", []Layer{LayerScene, LayerClip, LayerCallout, LayerMusic}},
		{"combat", []Layer{LayerCombat, LayerDice, LayerTimer, LayerMusic}},
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
