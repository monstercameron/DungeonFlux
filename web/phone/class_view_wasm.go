//go:build js && wasm

package phone

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func creationReadyLabel(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "es") {
		return "Listo"
	}
	return "Ready"
}

func creationLockedLabel(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "es") {
		return "Héroe bloqueado"
	}
	return "Hero locked"
}

func creationHint(locale string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "es") {
		return "Elige especie, género y clase para crear tu héroe."
	}
	return "Choose a species, gender, and class to roll your hero."
}

func creationClassPicker(model *CreationModel, refresh stateCounter, locale, selected string, disabled bool) ui.Node {
	choices := make([]ui.Node, 0, len(creationClassCopies))
	for _, option := range CreationClassesForLocale(locale) {
		choice := option
		tap := ui.UseEvent(func() {
			if !disabled {
				_ = model.SelectClass(choice.ID)
				refresh.Set(refresh.Get() + 1)
			}
		})
		selectedChoice := selected == choice.ID
		buttonStyle := map[string]string{
			"min-height": "76px", "min-width": "0", "width": "100%", "padding": ".55rem .5rem",
			"display": "flex", "gap": ".5rem", "align-items": "center", "text-align": "left",
			"border-radius": "10px", "border": "1px solid #4b4b4c", "background": "#1a1d26",
			"color": "#efe6d2", "font-size": ".9rem", "opacity": "1",
		}
		if selectedChoice {
			buttonStyle["border-color"] = "#d9a441"
			buttonStyle["background"] = "#332a19"
			buttonStyle["box-shadow"] = "inset 0 0 0 1px #d9a441"
		}
		if disabled {
			buttonStyle["opacity"] = ".72"
		}
		crest := html.Span(html.Props{Role: "img", Aria: map[string]string{"label": choice.Label + " crest"}, Style: map[string]string{
			"width": "32px", "height": "32px", "flex": "0 0 32px", "display": "flex", "align-items": "center", "justify-content": "center",
			"border": "1px solid #80652b", "border-radius": "50%", "color": "#d9a441", "font-size": "1.05rem", "font-family": "Georgia, serif",
		}}, html.Text(choice.Crest))
		copy := html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": ".15rem"}},
			html.Span(html.Props{Style: map[string]string{"font-weight": "700", "line-height": "1.1"}}, html.Text(choice.Label)),
			html.Span(html.Props{Style: map[string]string{"color": "#a89f8c", "font-size": ".73rem", "line-height": "1.2", "overflow-wrap": "anywhere"}}, html.Text(choice.Role)),
		)
		choices = append(choices, html.Button(html.Props{Type: "button", OnClick: tap, Disabled: disabled, Aria: map[string]string{"pressed": strconv.FormatBool(selectedChoice)}, Style: buttonStyle}, crest, copy))
	}
	return html.Fieldset(html.Props{Style: map[string]string{"max-width": "34rem", "min-width": "0", "width": "100%", "box-sizing": "border-box", "margin": "0 auto", "padding": ".8rem", "border": "1px solid #3a3a42", "border-radius": "12px", "background": "#171a23"}},
		html.Legend(html.Props{Style: map[string]string{"padding": "0 .35rem", "color": "#efe6d2", "font-weight": "700"}}, html.Text("Class")),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "display": "grid", "grid-template-columns": "repeat(2, minmax(0, 1fr))", "gap": ".55rem"}}, choices...))
}
