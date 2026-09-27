//go:build js && wasm

package phone

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// ChoiceRow renders a large, thumb-friendly legal move row.
func ChoiceRow(row ChoiceRowModel, tap ui.Handler) ui.Node {
	theme := DefaultPhoneTheme()
	style := map[string]string{
		"width": "100%", "min-height": "52px", "box-sizing": "border-box", "display": "flex",
		"align-items": "center", "gap": "12px", "padding": "10px 14px", "border-radius": "10px",
		"border": "1px solid rgba(168,159,140,.48)", "background": "rgba(23,26,35,.9)",
		"color": theme.Parchment, "font-family": theme.Serif, "font-size": "17px", "text-align": "left",
		"line-height": "1.18", "touch-action": "manipulation", "transition": "border-color " + theme.Transition,
	}
	if row.Highlighted {
		style["border"] = "1px solid " + theme.GoldBright
		style["box-shadow"] = "inset 0 0 14px rgba(217,164,65,.16), 0 0 12px rgba(217,164,65,.12)"
	}
	if !row.Enabled {
		style["color"] = theme.Muted
		style["opacity"] = ".66"
	}
	children := []ui.Node{choiceIcon(row.Icon)}
	text := html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": "3px"}}, html.Text(row.Label))
	if reason := ChoiceRowReason(row); !row.Enabled && reason != "" {
		text = html.Span(html.Props{Style: map[string]string{"min-width": "0", "display": "flex", "flex-direction": "column", "gap": "3px"}}, html.Text(row.Label), html.Small(html.Props{Style: map[string]string{"font-family": theme.Sans, "font-size": "11px", "color": theme.Muted}}, html.Text(reason)))
	}
	children = append(children, text)
	return html.Button(html.Props{Type: "button", Class: choiceClass(row), Disabled: !row.Enabled, OnClick: tap, Aria: map[string]string{"label": choiceLabel(row)}}, children...)
}

// PrimaryButton renders the gold outlined action used for roll, continue, and take.
func PrimaryButton(label string, tap ui.Handler, disabled bool) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Button(html.Props{Type: "button", Class: "df-phone-primary", Disabled: disabled, OnClick: tap, Style: map[string]string{
		"width": "100%", "min-height": "56px", "padding": "10px 22px", "border": "1px solid " + theme.GoldBright,
		"border-radius": "7px", "background": "linear-gradient(180deg, rgba(64,48,25,.92), rgba(25,22,19,.98))",
		"box-shadow": "inset 0 0 15px rgba(217,164,65,.14), 0 5px 16px rgba(0,0,0,.28)",
		"color":      theme.Parchment, "font-family": theme.Serif, "font-size": "20px", "touch-action": "manipulation",
	}}, html.Text(label))
}

// SecondaryButton renders a muted outlined action for non-primary choices.
func SecondaryButton(label string, tap ui.Handler, disabled bool) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Button(html.Props{Type: "button", Class: "df-phone-secondary", Disabled: disabled, OnClick: tap, Style: map[string]string{
		"width": "100%", "min-height": "52px", "padding": "9px 18px", "border": "1px solid rgba(168,159,140,.56)",
		"border-radius": "7px", "background": "rgba(15,17,23,.82)", "color": theme.Parchment,
		"font-family": theme.Serif, "font-size": "17px", "touch-action": "manipulation",
	}}, html.Text(label))
}

// NarrationCard renders the DM portrait, speaker name, and latest line.
func NarrationCard(speaker, body, portraitURL string) ui.Node {
	theme := DefaultPhoneTheme()
	portrait := html.Div(html.Props{Class: "df-phone-narration-portrait", Style: map[string]string{
		"width": "56px", "height": "56px", "flex": "0 0 56px", "display": "grid", "place-items": "center",
		"overflow": "hidden", "border": "1px solid " + theme.Gold, "border-radius": "50%", "background": theme.PanelRaised,
	}}, html.Text(initials(speaker)))
	if strings.TrimSpace(portraitURL) != "" {
		portrait = html.Div(html.Props{Class: "df-phone-narration-portrait", Style: map[string]string{
			"width": "56px", "height": "56px", "flex": "0 0 56px", "overflow": "hidden", "border": "1px solid " + theme.Gold, "border-radius": "50%", "background": theme.PanelRaised,
		}}, html.Img(html.Props{Src: portraitURL, Alt: speaker, Style: map[string]string{"width": "100%", "height": "100%", "object-fit": "cover"}}))
	}
	return html.Section(html.Props{Class: "df-phone-narration", Aria: map[string]string{"label": speaker}, Style: map[string]string{
		"display": "flex", "gap": "12px", "align-items": "flex-start", "padding": "12px", "border": "1px solid rgba(217,164,65,.52)",
		"border-radius": theme.BorderRadius, "background": "rgba(18,22,29,.92)", "box-shadow": "inset 0 0 22px rgba(217,164,65,.06)",
	}}, portrait, html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.Strong(html.Props{Style: map[string]string{"display": "block", "margin-bottom": "4px", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "16px"}}, html.Text(speaker)), html.P(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "17px", "font-style": "italic", "line-height": "1.3"}}, html.Text(body))))
}

// ResultBanner renders the pointed success or failure status plate.
func ResultBanner(result ResultBannerModel) ui.Node {
	theme := DefaultPhoneTheme()
	color, label := theme.Blood, "Failure"
	if result.Success {
		color, label = theme.Teal, "Success"
	}
	message := strings.TrimSpace(result.Message)
	if message == "" {
		message = label
	}
	return html.Div(html.Props{Class: "df-phone-result-banner df-phone-result-" + strings.ToLower(label), Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{
		"min-height": "44px", "display": "grid", "place-items": "center", "padding": "7px 20px", "border": "1px solid " + color,
		"border-radius": "5px", "background": "rgba(10,15,20,.82)", "box-shadow": "inset 0 0 14px " + color + "44", "color": color,
		"font-family": theme.Serif, "font-size": "20px", "font-weight": "600", "clip-path": "polygon(8px 0, calc(100% - 8px) 0, 100% 50%, calc(100% - 8px) 100%, 8px 100%, 0 50%)",
	}}, html.Text(message))
}

// StatRow renders the six compact ability cells from the character sheet.
func StatRow(stats []StatValue) ui.Node {
	theme := DefaultPhoneTheme()
	children := make([]ui.Node, 0, len(stats))
	for _, stat := range stats {
		modifier := strconv.Itoa(int(stat.Modifier))
		if stat.Modifier >= 0 {
			modifier = "+" + modifier
		}
		children = append(children, html.Div(html.Props{Class: "df-phone-stat", Style: map[string]string{"min-width": "0", "display": "grid", "gap": "2px", "padding": "7px 2px", "border-right": "1px solid rgba(168,159,140,.22)", "text-align": "center"}}, html.Small(html.Props{Style: map[string]string{"color": theme.Muted, "font-family": theme.Sans, "font-size": "10px", "letter-spacing": ".04em"}}, html.Text(stat.Name)), html.Strong(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "17px", "color": theme.Parchment}}, html.Text(strconv.Itoa(int(stat.Score)))), html.Span(html.Props{Style: map[string]string{"color": theme.GoldBright, "font-size": "12px"}}, html.Text(modifier))))
	}
	return html.Div(html.Props{Class: "df-phone-stat-row", Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(6, minmax(0, 1fr))", "border-top": "1px solid rgba(217,164,65,.42)", "border-bottom": "1px solid rgba(217,164,65,.42)"}}, children...)
}

// HPBar renders a labeled, teal character health bar.
func HPBar(hp, maximum int32) ui.Node {
	theme := DefaultPhoneTheme()
	percent := HPPercent(hp, maximum)
	return html.Div(html.Props{Class: "df-phone-hp", Aria: map[string]string{"label": "Hit points " + strconv.Itoa(int(hp)) + " of " + strconv.Itoa(int(maximum))}, Style: map[string]string{"display": "grid", "gap": "5px"}}, html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "font-family": theme.Sans, "font-size": "12px", "color": theme.Muted}}, html.Span(html.Props{}, html.Text(T("en", "phone.hp", nil))), html.Strong(html.Props{Style: map[string]string{"color": theme.Parchment}}, html.Text(strconv.Itoa(int(hp))+"/"+strconv.Itoa(int(maximum))))), html.Div(html.Props{Style: map[string]string{"height": "9px", "overflow": "hidden", "border-radius": "8px", "background": "#252a31"}}, html.Div(html.Props{Style: map[string]string{"width": strconv.Itoa(int(percent)) + "%", "height": "100%", "border-radius": "8px", "background": theme.Teal, "box-shadow": "0 0 10px rgba(47,181,154,.55)"}})))
}

// PortraitHero renders the scene image with its readable bottom gradient.
func PortraitHero(imageURL, name, role, quote string) ui.Node {
	theme := DefaultPhoneTheme()
	style := map[string]string{"position": "relative", "min-height": "280px", "overflow": "hidden", "border-radius": theme.BorderRadius, "border": "1px solid rgba(217,164,65,.5)", "background": theme.PanelRaised}
	if strings.TrimSpace(imageURL) != "" {
		style["background"] = "linear-gradient(180deg, transparent 34%, rgba(7,10,15,.96) 100%), url(\"" + imageURL + "\") center top / cover"
	}
	return html.Section(html.Props{Class: "df-phone-portrait-hero", Style: style}, html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "16px", "right": "16px", "bottom": "14px"}}, html.H1(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "26px"}}, html.Text(name)), html.P(html.Props{Style: map[string]string{"margin": "1px 0 5px", "color": theme.GoldBright, "font-family": theme.Sans, "font-size": "15px"}}, html.Text(role)), html.P(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px", "font-style": "italic", "line-height": "1.3"}}, html.Text(quote))))
}

// IconHeader renders the title medallion used by check and exploration screens.
func IconHeader(icon, title, subtitle string) ui.Node {
	theme := DefaultPhoneTheme()
	var mark ui.Node = html.Text(iconText(icon))
	if url := ArtURL(icon); url != "" {
		mark = html.Img(html.Props{Src: url, Alt: "", Style: map[string]string{"width": "34px", "height": "34px", "object-fit": "contain"}})
	}
	return html.Div(html.Props{Class: "df-phone-icon-header", Style: map[string]string{"display": "flex", "gap": "14px", "align-items": "center"}}, html.Div(html.Props{Style: map[string]string{"width": "56px", "height": "56px", "flex": "0 0 56px", "display": "grid", "place-items": "center", "border": "1px solid " + theme.Gold, "border-radius": "50%", "background": "rgba(40,34,25,.9)", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "27px", "box-shadow": "inset 0 0 14px rgba(217,164,65,.16)"}}, mark), html.Div(html.Props{Style: map[string]string{"min-width": "0"}}, html.H1(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "26px", "line-height": "1.1"}}, html.Text(title)), html.P(html.Props{Style: map[string]string{"margin": "4px 0 0", "color": theme.Muted, "font-family": theme.Serif, "font-size": "17px"}}, html.Text(subtitle))))
}

func choiceIcon(icon string) ui.Node {
	theme := DefaultPhoneTheme()
	if url := ArtURL(icon); url != "" {
		return html.Img(html.Props{Src: url, Alt: "", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": "30px", "height": "30px", "flex": "0 0 30px", "object-fit": "contain"}})
	}
	icon = iconText(icon)
	return html.Span(html.Props{Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"width": "30px", "height": "30px", "flex": "0 0 30px", "display": "grid", "place-items": "center", "border": "1px solid rgba(217,164,65,.42)", "border-radius": "50%", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "17px"}}, html.Text(icon))
}

func choiceClass(row ChoiceRowModel) string {
	class := "df-phone-choice-row"
	if row.Highlighted {
		class += " df-phone-choice-highlighted"
	}
	if !row.Enabled {
		class += " df-phone-choice-disabled"
	}
	return class
}

func choiceLabel(row ChoiceRowModel) string {
	if row.Enabled {
		return row.Label
	}
	if reason := ChoiceRowReason(row); reason != "" {
		return row.Label + ". " + reason
	}
	return row.Label
}

func initials(value string) string {
	for _, r := range strings.TrimSpace(value) {
		return string(r)
	}
	return "?"
}

// iconText turns an icon selector into a glyph for when its art is not loaded:
// asset names such as "ui/icon_talk" must never render as text.
func iconText(icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" || strings.Contains(icon, "/") || len([]rune(icon)) > 2 {
		return "◆"
	}
	return icon
}
