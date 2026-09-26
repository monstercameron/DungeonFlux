//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// DiceScreen renders the touch-first persuasion roll card.
func DiceScreen(model *DiceModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		roll := ui.UseEvent(func() {
			go func() { model.ApplyAct(<-model.Roll(context.Background())); refresh.Set(refresh.Get() + 1) }()
		})
		snapshot := model.Snapshot()
		result := "Roll Persuasion"
		if snapshot.Phase == DiceRolling {
			result = "Rolling…"
		}
		if snapshot.Phase == DiceResolved {
			result = "Result: " + number(snapshot.D20) + " " + snapshot.Outcome
		}
		if snapshot.Error != "" {
			result = snapshot.Error
		}
		label := "Persuasion " + signed(snapshot.Modifier) + " vs DC " + number(snapshot.DC)
		return html.Main(html.Props{Class: "df-phone df-phone-dice"},
			html.H1(html.Props{}, html.Text("Make your case")),
			html.P(html.Props{}, html.Text(label)),
			html.P(html.Props{Role: "status", Aria: map[string]string{"live": "polite"}}, html.Text(result)),
			html.Button(html.Props{Type: "button", OnClick: roll, Disabled: !snapshot.CanRoll}, html.Text("Roll d20")),
		)
	}
}

func signed(value int32) string {
	if value >= 0 {
		return "+" + number(value)
	}
	return "-" + number(-value)
}
