//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"strconv"
)

// EndScreen renders the outcome, final character state, thanks, and attribution.
func EndScreen(model *EndModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		state := model.Snapshot()
		locale := state.Locale
		if locale == "" {
			locale = "en"
		}
		return ui.CreateElement(func() ui.Node { return endPage(locale, state) })
	}
}

func endPage(locale string, state EndSnapshot) ui.Node {
	portrait := html.Div(html.Props{Class: "df-phone-end-portrait"}, html.Text(endInitials(state.Name)))
	if state.PortraitURL != "" {
		portrait = html.Div(html.Props{Class: "df-phone-end-portrait"}, html.Img(html.Props{Src: state.PortraitURL, Alt: state.Name}))
	}
	finalText := ""
	if state.FinalHPMax > 0 {
		finalText = strconv.Itoa(int(state.FinalHP)) + "/" + strconv.Itoa(int(state.FinalHPMax)) + " HP"
	}
	states := joinEndStates(state.FinalStates)
	if states != "" {
		if finalText != "" {
			finalText += " · "
		}
		finalText += states
	}
	return html.Main(html.Props{Class: "df-phone df-phone-end", Style: endPageStyle()},
		html.P(html.Props{Class: "df-phone-end-kicker"}, html.Text(endLabel(locale))),
		html.H1(html.Props{Class: "df-phone-end-outcome"}, html.Text(EndOutcome(locale, state.Outcome))),
		html.Section(html.Props{Class: "df-phone-end-hero", Aria: map[string]string{"label": endCharacterLabel(locale)}}, portrait, html.Div(html.Props{Class: "df-phone-end-character"}, html.H2(html.Props{}, html.Text(state.Name)), html.P(html.Props{}, html.Text(state.Class)), html.P(html.Props{Class: "df-phone-end-final"}, html.Text(finalText)))),
		html.P(html.Props{Class: "df-phone-end-thanks"}, html.Text(endThanks(locale))),
		html.P(html.Props{Class: "df-phone-end-attribution"}, html.Text(endAttribution(locale)+" "), html.A(html.Props{Href: srdAttributionURL, Target: "_blank", Rel: "noreferrer"}, html.Text(endSourceLabel(locale)))),
	)
}

func endPageStyle() map[string]string {
	return map[string]string{"background": "#10131b", "color": "#efe6d2", "min-height": "100vh", "box-sizing": "border-box", "padding": "clamp(24px, 7vw, 44px) 20px", "font-family": "system-ui, -apple-system, sans-serif", "text-align": "center", "overflow-wrap": "anywhere"}
}
func endInitials(name string) string {
	for _, r := range name {
		return string(r)
	}
	return "?"
}
func joinEndStates(values []string) string {
	result := ""
	for _, value := range values {
		if value == "" {
			continue
		}
		if result != "" {
			result += ", "
		}
		result += value
	}
	return result
}
func endLabel(locale string) string { return localizedSheet(locale, "THE END", "EL FINAL") }
func endCharacterLabel(locale string) string {
	return localizedSheet(locale, "Final character state", "Estado final del personaje")
}
func endThanks(locale string) string {
	return localizedSheet(locale, "Thanks for playing.", "Gracias por jugar.")
}
func endAttribution(locale string) string {
	return localizedSheet(locale, "Rules material from the System Reference Document 5.2.1 by Wizards of the Coast LLC, licensed under CC-BY 4.0.", "Material de reglas del System Reference Document 5.2.1 de Wizards of the Coast LLC, bajo licencia CC-BY 4.0.")
}
func endSourceLabel(locale string) string {
	return localizedSheet(locale, "Read the SRD", "Leer el SRD")
}
