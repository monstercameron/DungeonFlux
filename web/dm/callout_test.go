package dm

import (
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestNewCalloutView_HidesEmptyText(t *testing.T) {
	if got := NewCalloutView("  "); got.Visible || got.Text != "" {
		t.Fatalf("empty callout = %#v", got)
	}
}

func TestNewCalloutView_TrimsPresentationCopy(t *testing.T) {
	if got := NewCalloutView("  DM steering: hook  "); got.Text != "DM steering: hook" || !got.Visible {
		t.Fatalf("trimmed callout = %#v", got)
	}
}

func TestNewCalloutView_ShowsText(t *testing.T) {
	if got := NewCalloutView("DM steering: hook"); !got.Visible || got.Text != "DM steering: hook" {
		t.Fatalf("callout = %#v", got)
	}
}

func TestCalloutViewFromDMView_ProjectsText(t *testing.T) {
	if got := CalloutViewFromDMView(nil); got.Visible {
		t.Fatal("nil DM view should hide callout")
	}
	if got := CalloutViewFromDMView(&dungeonfluxv1.DMView{Callout: "Hook"}); !got.Visible || got.Text != "Hook" {
		t.Fatalf("dm callout = %#v", got)
	}
}
