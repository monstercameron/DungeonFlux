//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
)

// SheetScreen renders the compact player sheet for a portrait phone display.
func SheetScreen(model *SheetModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		state := model.Snapshot()
		portrait := html.Div(html.Props{Class: "df-phone-portrait", Role: "img", Aria: map[string]string{"label": state.Name}}, html.Text(state.Name))
		hp := html.P(html.Props{}, html.Text(hpLabel(state)))
		conditions := html.P(html.Props{}, html.Text(conditionLabel(state)))
		return html.Main(html.Props{Class: "df-phone df-phone-sheet"},
			html.H1(html.Props{}, html.Text(model.Summary())), portrait, hp, conditions,
			html.P(html.Props{Role: "status"}, html.Text(state.StatusText)),
		)
	}
}

func hpLabel(state SheetSnapshot) string {
	if state.HPMax <= 0 {
		return "HP —"
	}
	return "HP " + number(state.HP) + "/" + number(state.HPMax)
}

func conditionLabel(state SheetSnapshot) string {
	if len(state.Conditions) == 0 {
		return "No conditions"
	}
	return "Conditions: " + state.Conditions[0]
}

func number(value int32) string {
	if value == 0 {
		return "0"
	}
	if value < 0 {
		return "-" + number(-value)
	}
	result := ""
	for value > 0 {
		result = string(rune('0'+value%10)) + result
		value /= 10
	}
	return result
}
