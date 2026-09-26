//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// MovesScreen renders the legal action menu for a portrait phone display.
func MovesScreen(model *MovesModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		current := model.Snapshot()
		locale := current.Locale
		if locale == "" {
			locale = "en"
		}
		children := make([]ui.Node, 0, len(current.Moves)*2+2)
		for _, move := range current.Moves {
			item := move
			tap := ui.UseEvent(func() {
				go func() { model.ApplyAct(<-model.Tap(context.Background(), item.ID)); refresh.Set(refresh.Get() + 1) }()
			})
			children = append(children, html.Button(html.Props{Type: "button", Class: moveClass(item), OnClick: tap, Disabled: !item.Enabled}, html.Text(item.Label)))
			if !item.Enabled && item.Reason != "" {
				children = append(children, html.P(html.Props{Class: "df-phone-move-reason", Role: "note"}, html.Text(item.Reason)))
			}
		}
		if current.Error != "" {
			children = append(children, html.P(html.Props{Role: "alert"}, html.Text(current.Error)))
		}
		children = append([]ui.Node{html.H1(html.Props{}, html.Text(MovesTitle(locale)))}, children...)
		return html.Main(html.Props{Class: "df-phone df-phone-moves"}, children...)
	}
}

func moveClass(move MoveSnapshot) string {
	if move.Enabled {
		return "df-phone-move"
	}
	return "df-phone-move df-phone-move-disabled"
}
