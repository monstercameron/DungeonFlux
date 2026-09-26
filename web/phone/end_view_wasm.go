//go:build js && wasm

package phone

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

const srdAttributionURL = "https://www.dndbeyond.com/srd"

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
	portrait := endPortrait(state)
	finalText := ""
	if state.FinalHPMax > 0 {
		finalText = strconv.Itoa(int(state.FinalHP)) + "/" + strconv.Itoa(int(state.FinalHPMax)) + " HP"
	}
	return html.Section(html.Props{Class: "df-phone-end", Role: "main", Style: endPageStyle()},
		html.Div(html.Props{Style: map[string]string{"padding": "18px 8px 8px", "text-align": "center"}},
			html.P(html.Props{Class: "df-phone-end-kicker", Style: map[string]string{"margin": "0 0 5px", "color": "#d9a441", "font-size": "10px", "font-weight": "700", "letter-spacing": ".25em"}}, html.Text(endLabel(locale))),
			html.H1(html.Props{Class: "df-phone-end-outcome", Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-family": "Cormorant Garamond, Cinzel, Georgia, serif", "font-size": "30px", "line-height": "1.05"}}, html.Text(EndOutcome(locale, state.Outcome))),
		),
		ResultBanner(ResultBannerModel{Success: true, Message: "The story continues"}),
		html.Section(html.Props{Class: "df-phone-end-hero", Aria: map[string]string{"label": endCharacterLabel(locale)}, Style: map[string]string{"display": "flex", "gap": "13px", "align-items": "center", "padding": "12px", "border": "1px solid rgba(217,164,65,.52)", "border-radius": "12px", "background": "linear-gradient(135deg, rgba(43,36,30,.96), rgba(17,21,29,.98))", "box-shadow": "inset 0 0 22px rgba(217,164,65,.07)"}}, portrait, html.Div(html.Props{Class: "df-phone-end-character", Style: map[string]string{"min-width": "0"}}, html.H2(html.Props{Style: map[string]string{"margin": "0 0 3px", "color": "#e7c27a", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "24px"}}, html.Text(state.Name)), html.P(html.Props{Style: map[string]string{"margin": "0 0 8px", "color": "#a89f8c", "font-size": "12px", "letter-spacing": ".08em", "text-transform": "uppercase"}}, html.Text(state.Class)), html.P(html.Props{Class: "df-phone-end-final", Style: map[string]string{"margin": "0", "color": "#efe6d2", "font-size": "13px"}}, html.Text(finalText)))),
		endFinalStates(state.FinalStates),
		html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "padding": "15px 5px 6px", "text-align": "center"}}, html.P(html.Props{Class: "df-phone-end-thanks", Style: map[string]string{"margin": "0 0 5px", "color": "#efe6d2", "font-family": "Cormorant Garamond, Georgia, serif", "font-size": "19px"}}, html.Text(endThanks(locale))), html.P(html.Props{Class: "df-phone-end-attribution", Style: map[string]string{"margin": "0", "color": "#777467", "font-size": "10px", "line-height": "1.35"}}, html.Text(endAttribution(locale)+" "), html.A(html.Props{Href: srdAttributionURL, Target: "_blank", Rel: "noreferrer", Style: map[string]string{"color": "#d9a441"}}, html.Text(endSourceLabel(locale))))),
	)
}

func endPortrait(state EndSnapshot) ui.Node {
	style := map[string]string{"width": "82px", "height": "104px", "flex": "0 0 82px", "display": "grid", "place-items": "center", "overflow": "hidden", "border": "1px solid #d9a441", "border-radius": "8px", "background": "radial-gradient(circle, #354052, #171a23 70%)", "color": "#e7c27a", "font-family": "Georgia, serif", "font-size": "25px"}
	if state.PortraitURL == "" {
		return html.Div(html.Props{Class: "df-phone-end-portrait", Role: "img", Aria: map[string]string{"label": state.Name}, Style: style}, html.Text(endInitials(state.Name)))
	}
	return html.Div(html.Props{Class: "df-phone-end-portrait", Style: style}, html.Img(html.Props{Src: state.PortraitURL, Alt: state.Name, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
}

func endFinalStates(values []string) ui.Node {
	if len(values) == 0 {
		return nil
	}
	items := make([]ui.Node, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		items = append(items, html.Span(html.Props{Style: map[string]string{"padding": "5px 9px", "border": "1px solid rgba(47,181,154,.55)", "border-radius": "14px", "color": "#8dd1c9", "font-size": "11px"}}, html.Text(value)))
	}
	if len(items) == 0 {
		return nil
	}
	return html.Div(html.Props{Style: map[string]string{"display": "flex", "flex-wrap": "wrap", "justify-content": "center", "gap": "6px", "padding": "2px 0"}}, items...)
}

func endPageStyle() map[string]string {
	return map[string]string{"min-height": "100%", "box-sizing": "border-box", "display": "flex", "flex-direction": "column", "gap": "12px", "padding": "4px 0 2px", "background": "radial-gradient(circle at 50% 0%, #2c2b33 0, #161b25 40%, #0b0f16 100%)", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif", "overflow-wrap": "anywhere"}
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
