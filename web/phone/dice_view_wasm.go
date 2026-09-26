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
		locale := snapshot.Locale
		if locale == "" {
			locale = "en"
		}
		result := DiceResult(locale, snapshot.Phase, snapshot.D20, snapshot.Outcome)
		if snapshot.Error != "" {
			result = snapshot.Error
		}
		label := DiceCheckLabel(locale, snapshot.Modifier, snapshot.DC)
		phaseClass := "df-phone-dice-phase-" + string(snapshot.Phase)
		cardStyle := map[string]string{
			"min-height": "100dvh", "box-sizing": "border-box", "display": "flex", "flex-direction": "column",
			"gap": "1rem", "padding": "clamp(1.25rem, 5vw, 2rem)", "background": "radial-gradient(circle at 50% 18%, #292d3b 0, #171923 48%, #0f1117 100%)",
			"color": "#efe6d2", "font-family": "system-ui, sans-serif", "text-align": "center",
		}
		dieStyle := map[string]string{
			"display": "grid", "place-items": "center", "width": "clamp(9rem, 42vw, 12rem)", "height": "clamp(9rem, 42vw, 12rem)",
			"margin": "1rem auto", "border": "2px solid #d9a441", "border-radius": "24px", "background": "linear-gradient(145deg, #343342, #1b1c27)",
			"box-shadow": "0 14px 40px rgba(0,0,0,.4), inset 0 0 0 1px rgba(239,230,210,.12)", "font-family": "Georgia, serif", "font-size": "clamp(4rem, 20vw, 6rem)", "font-weight": "700",
		}
		buttonStyle := map[string]string{
			"width": "100%", "min-height": "56px", "margin-top": "auto", "border": "1px solid #e7bc5a", "border-radius": "12px", "background": "#d9a441", "color": "#171923", "font-size": "1.05rem", "font-weight": "750", "letter-spacing": ".02em", "touch-action": "manipulation",
		}
		return html.Main(html.Props{Class: "df-phone df-phone-dice " + phaseClass, Style: cardStyle, Role: "main", Aria: map[string]string{"label": DiceTitle(locale)}},
			html.P(html.Props{Class: "df-phone-dice-eyebrow", Style: map[string]string{"margin": "0", "color": "#d9a441", "font-size": ".76rem", "font-weight": "700", "letter-spacing": ".16em", "text-transform": "uppercase"}}, html.Text("Persuasion check")),
			html.H1(html.Props{Style: map[string]string{"margin": "0", "font-family": "Georgia, serif", "font-size": "clamp(1.9rem, 8vw, 2.6rem)", "line-height": "1.1"}}, html.Text(DiceTitle(locale))),
			html.P(html.Props{Class: "df-phone-dice-check", Style: map[string]string{"margin": "0", "color": "#a89f8c", "font-size": "1.05rem"}}, html.Text(label)),
			html.Div(html.Props{Class: "df-phone-dice-face", Role: "img", Aria: map[string]string{"label": result}, Style: dieStyle}, html.Text(DiceFace(snapshot.Phase, snapshot.D20))),
			html.P(html.Props{Class: "df-phone-dice-result", Role: "status", Aria: map[string]string{"live": "polite", "atomic": "true"}, Style: map[string]string{"min-height": "2.8rem", "margin": "0", "font-size": "1rem", "line-height": "1.45"}}, html.Text(result)),
			html.Button(html.Props{Type: "button", Class: "df-phone-dice-roll", OnClick: roll, Disabled: !snapshot.CanRoll, Style: buttonStyle}, html.Text(DiceButton(locale))),
		)
	}
}
