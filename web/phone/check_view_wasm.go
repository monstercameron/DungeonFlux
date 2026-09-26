//go:build js && wasm

package phone

import (
	"context"
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// CheckScreen renders the offer, roll, and resolved states of a phone check.
func CheckScreen(model *DiceModel) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		snapshot := model.Snapshot()
		presentation := CheckPresentationFromDice(snapshot)
		roll := ui.UseEvent(func() {
			go func() {
				model.ApplyAct(<-model.Roll(context.Background()))
				refresh.Set(refresh.Get() + 1)
			}()
		})
		content := checkOffer(snapshot, presentation, roll)
		if snapshot.Phase == DiceResolved {
			content = checkResult(presentation)
		}
		return html.Main(html.Props{Class: "df-phone df-phone-dice df-phone-check", Role: "main", Aria: map[string]string{"label": presentation.Name}, Style: checkScreenStyle()}, content)
	}
}

func checkScreenStyle() map[string]string {
	return map[string]string{
		"min-height": "100%", "box-sizing": "border-box", "display": "flex", "flex-direction": "column",
		"gap": "14px", "padding": "4px 0 10px", "color": "#efe6d2", "font-family": "Inter, ui-sans-serif, system-ui, sans-serif",
		"background": diceBackground(), "background-size": "cover", "background-position": "center",
	}
}

func checkOffer(snapshot DiceSnapshot, presentation CheckPresentation, roll ui.Handler) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Div(html.Props{Class: "df-phone-check-offer", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "14px", "min-height": "100%"}},
		IconHeader(checkIcon("ui/icon_persuade", "◇"), presentation.Name, presentation.Ability+" check"),
		html.P(html.Props{Class: "df-phone-check-description", Style: map[string]string{"margin": "0", "padding": "0 4px", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "17px", "font-style": "italic", "line-height": "1.35", "text-align": "center"}}, html.Text(presentation.Description)),
		checkModifierPanel(presentation),
		html.Div(html.Props{Style: map[string]string{"margin-top": "auto", "padding-top": "4px"}}, PrimaryButton("◇  Roll", roll, !snapshot.CanRoll)),
	)
}

func checkModifierPanel(presentation CheckPresentation) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Section(html.Props{Class: "df-phone-check-modifier", Aria: map[string]string{"label": "Check modifier"}, Style: map[string]string{
		"display": "grid", "grid-template-columns": "1fr auto", "gap": "8px 16px", "align-items": "center", "padding": "14px 16px",
		"border": "1px solid rgba(217,164,65,.58)", "border-radius": "12px", "background": "rgba(18,22,29,.9)", "box-shadow": "inset 0 0 20px rgba(217,164,65,.06)",
	}},
		html.Div(html.Props{}, html.Strong(html.Props{Style: map[string]string{"display": "block", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "18px"}}, html.Text(presentation.Ability+" ("+presentation.Skill+")")), html.Span(html.Props{Style: map[string]string{"display": "block", "margin-top": "3px", "color": theme.Muted, "font-size": "13px"}}, html.Text("Roll d20 "+SignedModifier(presentation.Modifier)))),
		html.Strong(html.Props{Class: "df-phone-check-modifier-value", Style: map[string]string{"color": theme.GoldBright, "font-family": theme.Serif, "font-size": "28px"}}, html.Text(SignedModifier(presentation.Modifier))),
		html.Div(html.Props{Style: map[string]string{"grid-column": "1 / -1", "display": "flex", "justify-content": "space-between", "padding-top": "8px", "border-top": "1px solid rgba(168,159,140,.22)", "color": theme.Muted, "font-size": "13px"}}, html.Span(html.Props{}, html.Text(T("en", "phone.check.difficulty", nil))), html.Span(html.Props{}, html.Text("DC "+strconv.Itoa(int(presentation.DC))))),
	)
}

func checkResult(presentation CheckPresentation) ui.Node {
	theme := DefaultPhoneTheme()
	children := []ui.Node{
		IconHeader(checkIcon("ui/icon_persuade", "◇"), presentation.Name, presentation.Ability),
		html.P(html.Props{Class: "df-phone-check-quote", Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px", "font-style": "italic", "line-height": "1.35", "text-align": "center"}}, html.Text(presentation.Quote)),
		checkDie(presentation),
		html.Div(html.Props{Class: "df-phone-check-total", Style: map[string]string{"margin": "-4px 0 0", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "24px", "text-align": "center"}}, html.Text(checkTotalLabel(presentation))),
	}
	if presentation.HasOutcome {
		children = append(children, ResultBanner(ResultBannerModel{Success: presentation.Success, Message: presentation.Outcome}))
	}
	children = append(children,
		html.Div(html.Props{Class: "df-phone-check-result-text", Style: map[string]string{"padding": "13px 14px", "border": "1px solid rgba(168,159,140,.4)", "border-radius": "10px", "background": "rgba(18,22,29,.92)", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px", "line-height": "1.35"}}, html.Text(presentation.ResultText)),
		html.Div(html.Props{Style: map[string]string{"margin-top": "auto"}}, SecondaryButton("Continue", ui.Handler{}, false)),
	)
	return html.Div(html.Props{Class: "df-phone-check-result", Style: map[string]string{"display": "flex", "flex-direction": "column", "gap": "12px", "min-height": "100%"}}, children...)
}

func checkDie(presentation CheckPresentation) ui.Node {
	asset := d20Asset
	if presentation.HasOutcome {
		if presentation.Success {
			asset = d20SuccessAsset
		} else {
			asset = d20FailAsset
		}
	}
	style := map[string]string{"position": "relative", "width": "clamp(132px, 38vw, 176px)", "height": "clamp(132px, 38vw, 176px)", "margin": "0 auto", "display": "grid", "place-items": "center", "border-radius": "22px", "background": "linear-gradient(145deg, #343342, #1b1c27)", "box-shadow": "0 12px 30px rgba(0,0,0,.46)"}
	children := make([]ui.Node, 0, 2)
	if url := ArtURL(asset); url != "" {
		children = append(children, html.Img(html.Props{Src: url, Alt: "", Aria: map[string]string{"hidden": "true"}, Style: map[string]string{"position": "absolute", "inset": "0", "width": "100%", "height": "100%", "object-fit": "contain"}}))
	}
	if presentation.HasRoll {
		children = append(children, html.Strong(html.Props{Style: map[string]string{"position": "relative", "color": "#fff1d4", "font-family": "Georgia, serif", "font-size": "clamp(3rem, 16vw, 5rem)", "text-shadow": "0 2px 8px #000"}}, html.Text(strconv.Itoa(int(presentation.D20)))))
	}
	return html.Div(html.Props{Class: "df-phone-check-die", Role: "img", Aria: map[string]string{"label": "d20 result"}, Style: style}, children...)
}

func checkTotalLabel(presentation CheckPresentation) string {
	if !presentation.HasRoll {
		return "Total: —"
	}
	return "Total: " + strconv.Itoa(int(presentation.Total))
}

func checkIcon(asset, fallback string) string {
	if url := ArtURL(asset); url != "" {
		return asset
	}
	return fallback
}

func diceBackground() string {
	if url := ArtURL("ui/check_backdrop"); url != "" {
		return "linear-gradient(180deg, rgba(10, 13, 19, .2), rgba(10, 13, 19, .76)), url(\"" + url + "\")"
	}
	if url := ArtURL(phoneBackgroundAsset); url != "" {
		return "linear-gradient(180deg, rgba(10, 13, 19, .2), rgba(10, 13, 19, .78)), url(\"" + url + "\")"
	}
	return "radial-gradient(circle at 50% 18%, #292d3b 0, #171923 48%, #0f1117 100%)"
}
