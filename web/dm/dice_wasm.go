//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// DiceComponent renders a large, readable result for the DM display.
func DiceComponent(view DiceView) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(view.Locale)
		if view.State == "" {
			return html.Div(html.Props{Class: "df-dm-dice", Hidden: true})
		}
		presentation := PresentDice(view)
		class := "df-dm-dice " + presentation.StateClass
		return html.Section(html.Props{Class: class, Role: "status", Aria: map[string]string{"label": T(locale, "dm.dice_aria", nil)}, Style: diceStyle()},
			diceAnimationStyle(),
			html.Div(html.Props{Class: "df-dm-dice-kicker", Style: map[string]string{"color": "#d9a441", "font-family": "Georgia, serif", "font-size": "clamp(18px, 2vw, 32px)", "letter-spacing": "0.12em", "text-transform": "uppercase"}}, ui.Text(DiceHeading(locale, view.Kind))),
			html.Div(html.Props{Class: "df-dm-dice-main", Style: map[string]string{"display": "flex", "align-items": "center", "gap": "clamp(18px, 2vw, 36px)"}},
				html.Div(html.Props{Class: "df-dm-dice-face", Style: diceFaceStyle(presentation), Raw: map[string]any{"data-state": view.State}}, ui.Text(presentation.FaceLabel)),
				html.Div(html.Props{Class: "df-dm-dice-detail", Style: map[string]string{"display": "flex", "flex": "1 1 220px", "min-width": "0", "flex-direction": "column", "gap": "8px"}},
					html.P(html.Props{Class: "df-dm-dice-modifier", Style: map[string]string{"margin": "0", "font-size": "clamp(20px, 2.3vw, 40px)", "font-weight": "700", "color": "#efe6d2"}}, ui.Text(modifierText(locale, view))),
					diceOutcome(locale, view),
				),
			),
		)
	}
}

func diceStyle() map[string]string {
	return map[string]string{"box-sizing": "border-box", "display": "inline-flex", "flex-direction": "column", "width": "min(760px, calc(100vw - 24px))", "max-width": "calc(100vw - 24px)", "padding": "clamp(18px, 2vw, 34px) clamp(24px, 3vw, 52px)", "border": "1px solid rgba(217, 164, 65, 0.72)", "border-radius": "14px", "background": "linear-gradient(135deg, rgba(15, 17, 23, 0.96), rgba(26, 29, 38, 0.91))", "box-shadow": "0 16px 42px rgba(0, 0, 0, 0.42), inset 0 0 0 1px rgba(239, 230, 210, 0.08)", "font-family": "Arial, sans-serif", "pointer-events": "none"}
}

func diceAnimationStyle() ui.Node {
	return html.Tag("style", html.Props{ID: "df-dm-dice-animation"}, html.Text("@keyframes df-dice-roll{from{transform:rotate(-7deg) scale(.96)}to{transform:rotate(7deg) scale(1.04)}}@media (prefers-reduced-motion:reduce){.df-dm-dice-face{animation:none!important}}"))
}

func diceFaceStyle(presentation DicePresentation) map[string]string {
	style := map[string]string{"display": "grid", "place-items": "center", "flex": "0 0 clamp(104px, 10vw, 180px)", "height": "clamp(104px, 10vw, 180px)", "border": "3px solid #d9a441", "border-radius": "22px", "background": "radial-gradient(circle at 35% 28%, #fff8e7, #d9a441 70%, #8f641d)", "color": "#17130c", "font-family": "Georgia, serif", "font-size": "clamp(62px, 7vw, 126px)", "font-weight": "700", "line-height": "1", "box-shadow": "0 8px 22px rgba(0, 0, 0, 0.45), inset 0 2px 0 rgba(255,255,255,0.5)"}
	if presentation.IsRolling {
		style["animation"] = "df-dice-roll 650ms ease-in-out infinite alternate"
		style["color"] = "#3a2a11"
	}
	return style
}

func modifierText(locale string, view DiceView) string {
	text := "d20"
	if view.Modifier >= 0 {
		text += " + " + strconv.Itoa(int(view.Modifier))
	} else {
		text += " - " + strconv.Itoa(int(-view.Modifier))
	}
	if view.DC > 0 {
		text += " " + VsDC(locale, view.DC)
	}
	if view.VsLabel != "" {
		text += " " + view.VsLabel
	}
	return text
}

func diceOutcome(locale string, view DiceView) ui.Node {
	presentation := PresentDice(view)
	text := presentation.ResultText
	if view.Damage != nil && view.Damage.Total != 0 {
		text += " · " + strconv.Itoa(int(view.Damage.Total)) + " " + view.Damage.Type
	}
	if view.Crit {
		text = CritLabel(locale)
	}
	if text == "" {
		text = view.State
	}
	color := "#a89f8c"
	if presentation.IsSuccess || view.Crit {
		color = "#3aa39a"
	} else if presentation.IsFailure {
		color = "#b3372f"
	}
	return html.P(html.Props{Class: "df-dm-dice-outcome", Style: map[string]string{"margin": "0", "color": color, "font-size": "clamp(22px, 2.6vw, 48px)", "font-weight": "800", "letter-spacing": "0.04em", "text-transform": "uppercase"}}, ui.Text(text))
}

// TimerComponent renders the remaining turn time as a progress bar.
func TimerComponent(view TimerView) router.Component {
	return func(_ router.Attrs) *router.Element {
		locale := localeOrDefault(view.Locale)
		props := html.Props{Class: "df-dm-timer", Role: "timer", Aria: map[string]string{"label": timerLabel(locale, view)}}
		return html.Section(props,
			html.Div(html.Props{Class: "df-dm-timer-label"}, ui.Text(timerLabel(locale, view))),
			html.Progress(html.Props{Class: "df-dm-timer-bar", Raw: map[string]any{
				"max":   strconv.FormatInt(view.TotalMS, 10),
				"value": strconv.FormatInt(maxTimerMS(view.RemainingMS), 10),
			}}),
		)
	}
}

func timerLabel(locale string, view TimerView) string {
	if view.Frozen {
		return TimerPaused(locale)
	}
	return TimerLabel(locale, maxTimerMS(view.RemainingMS))
}

func maxTimerMS(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
