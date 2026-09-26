package dm

import dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"

// CalloutView is the browser-owned projection of the TV steering callout.
type CalloutView struct {
	Text    string
	Visible bool
}

// NewCalloutView creates a callout projection, hiding empty copy.
func NewCalloutView(text string) CalloutView {
	return CalloutView{Text: text, Visible: text != ""}
}

// CalloutViewFromDMView projects the steering callout from a DM snapshot.
func CalloutViewFromDMView(view *dungeonfluxv1.DMView) CalloutView {
	if view == nil {
		return CalloutView{}
	}
	return NewCalloutView(view.GetCallout())
}
