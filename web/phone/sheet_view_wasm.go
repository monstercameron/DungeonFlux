//go:build js && wasm

package phone

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// SheetScreen renders the compact player sheet for a portrait phone display.
func SheetScreen(model *SheetModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		state := model.Snapshot()
		locale := state.Locale
		if locale == "" {
			locale = "en"
		}
		return sheetPage(locale, state)
	}
}

func sheetPage(locale string, state SheetSnapshot) ui.Node {
	portrait := sheetPortrait(state)
	hpPercent := SheetHPPercent(state.HP, state.HPMax)
	hpText := SheetHP(locale, state.HP, state.HPMax)
	hp := html.Section(html.Props{Class: "df-phone-sheet-card df-phone-sheet-hp", Aria: map[string]string{"label": hpText}},
		html.Div(html.Props{Class: "df-phone-sheet-card-head"}, html.Span(html.Props{Class: "df-phone-sheet-label"}, html.Text(sheetHPLabel(locale))), html.Strong(html.Props{Class: "df-phone-sheet-value"}, html.Text(hpText))),
		html.Progress(html.Props{Class: "df-phone-hp-meter " + SheetHPClass(state.HP, state.HPMax), Value: strconv.Itoa(int(hpPercent)), Max: "100", Raw: map[string]any{"aria-label": hpText}}),
	)
	stats := html.Section(html.Props{Class: "df-phone-sheet-card df-phone-sheet-stats", Aria: map[string]string{"label": sheetStatsLabel(locale)}},
		html.Div(html.Props{Class: "df-phone-sheet-card-head"}, html.Span(html.Props{Class: "df-phone-sheet-label"}, html.Text(sheetStatsLabel(locale))), html.Strong(html.Props{Class: "df-phone-sheet-stat"}, html.Text(sheetModifier(state.PersuasionModifier)))),
		html.P(html.Props{Class: "df-phone-sheet-stat-caption"}, html.Text(sheetPersuasionLabel(locale))),
	)
	conditions := sheetConditions(locale, state.Conditions)
	hook := sheetHook(locale, state.Hook)
	status := state.StatusText
	if status == "" {
		status = sheetReadyLabel(locale)
	}
	return html.Main(html.Props{Class: "df-phone df-phone-sheet", Style: sheetPageStyle()},
		html.Header(html.Props{Class: "df-phone-sheet-header"}, portrait, sheetSpeciesBadge(state), sheetClassCrest(state), html.Div(html.Props{Class: "df-phone-sheet-title"}, html.H1(html.Props{}, html.Text(SheetName(locale, state.Name, state.Class))), html.P(html.Props{Class: "df-phone-sheet-kicker"}, html.Text(sheetKicker(locale))))),
		html.Div(html.Props{Class: "df-phone-sheet-grid"}, hp, stats),
		conditions,
		hook,
		html.P(html.Props{Class: "df-phone-sheet-status", Role: "status"}, html.Text(status)),
	)
}

func sheetPortrait(state SheetSnapshot) ui.Node {
	if state.PortraitURL == "" {
		if url := ArtURL(speciesArtAsset(state.Species)); url != "" {
			return html.Img(html.Props{Class: "df-phone-portrait", Src: url, Alt: state.Species})
		}
		return html.Div(html.Props{Class: "df-phone-portrait df-phone-portrait-fallback", Role: "img", Aria: map[string]string{"label": state.Name}}, html.Text(sheetInitials(state.Name)))
	}
	return html.Img(html.Props{Class: "df-phone-portrait", Src: state.PortraitURL, Alt: state.Name})
}

func sheetClassCrest(state SheetSnapshot) ui.Node {
	if url := ArtURL(classArtAsset(state.Class)); url != "" {
		return html.Img(html.Props{Class: "df-phone-class-crest", Src: url, Alt: state.Class, Style: map[string]string{"width": "46px", "height": "46px", "object-fit": "contain", "flex": "0 0 46px"}})
	}
	return html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"display": "none"}}, html.Text(""))
}

func sheetSpeciesBadge(state SheetSnapshot) ui.Node {
	if url := ArtURL(speciesArtAsset(state.Species)); url != "" {
		return html.Img(html.Props{Class: "df-phone-species-badge", Src: url, Alt: state.Species, Style: map[string]string{"width": "34px", "height": "42px", "object-fit": "cover", "border-radius": "6px", "flex": "0 0 34px"}})
	}
	return html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"display": "none"}}, html.Text(""))
}

func sheetConditions(locale string, values []string) ui.Node {
	items := make([]ui.Node, 0, len(values))
	for _, value := range values {
		if value != "" {
			icon := html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"display": "none"}}, html.Text(""))
			if url := ArtURL(statusArtAsset(value)); url != "" {
				icon = html.Img(html.Props{Src: url, Alt: "", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": "28px", "height": "28px", "object-fit": "contain"}})
			}
			items = append(items, html.Li(html.Props{Class: "df-phone-condition", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "8px"}}, icon, html.Text(value)))
		}
	}
	if len(items) == 0 {
		items = append(items, html.Li(html.Props{Class: "df-phone-condition df-phone-condition-clear"}, html.Text(sheetNoConditionsLabel(locale))))
	}
	return html.Section(html.Props{Class: "df-phone-sheet-card df-phone-sheet-conditions", Aria: map[string]string{"label": sheetConditionsLabel(locale)}},
		html.Div(html.Props{Class: "df-phone-sheet-card-head"}, html.Span(html.Props{Class: "df-phone-sheet-label"}, html.Text(sheetConditionsLabel(locale)))), html.Ul(html.Props{Class: "df-phone-condition-list"}, items...),
	)
}

func sheetHook(locale, hook string) ui.Node {
	if hook == "" {
		return nil
	}
	return html.Section(html.Props{Class: "df-phone-sheet-hook", Aria: map[string]string{"label": sheetHookLabel(locale)}}, html.P(html.Props{Class: "df-phone-sheet-hook-text"}, html.Text("["+hook+"]")))
}

func sheetPageStyle() map[string]string {
	background := "#10131b"
	if url := ArtURL(phoneBackgroundAsset); url != "" {
		background = "linear-gradient(180deg, rgba(10, 13, 19, .2), rgba(10, 13, 19, .82)), url(\"" + url + "\")"
	}
	return map[string]string{"background": background, "background-size": "cover", "background-position": "center", "color": "#efe6d2", "min-height": "100vh", "box-sizing": "border-box", "padding": "clamp(16px, 4vw, 28px)", "font-family": "system-ui, -apple-system, sans-serif"}
}

func sheetInitials(name string) string {
	for _, runeValue := range name {
		return string(runeValue)
	}
	return "?"
}

func sheetModifier(value int32) string {
	if value >= 0 {
		return "+" + strconv.Itoa(int(value))
	}
	return strconv.Itoa(int(value))
}

func sheetHPLabel(locale string) string    { return localizedSheet(locale, "HP", "PV") }
func sheetStatsLabel(locale string) string { return localizedSheet(locale, "YOUR EDGE", "TU VENTAJA") }
func sheetPersuasionLabel(locale string) string {
	return localizedSheet(locale, "Persuasion", "Persuasion")
}
func sheetConditionsLabel(locale string) string {
	return localizedSheet(locale, "CONDITIONS", "CONDICIONES")
}
func sheetNoConditionsLabel(locale string) string { return localizedSheet(locale, "None", "Ninguna") }
func sheetHookLabel(locale string) string         { return localizedSheet(locale, "YOUR STORY", "TU HISTORIA") }
func sheetKicker(locale string) string {
	return localizedSheet(locale, "PLAYER SHEET", "FICHA DEL JUGADOR")
}
func sheetReadyLabel(locale string) string {
	return localizedSheet(locale, "Ready for the next move.", "Listo para el siguiente movimiento.")
}
func localizedSheet(locale, english, spanish string) string {
	if locale == "es" {
		return spanish
	}
	return english
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
