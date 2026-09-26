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
		if view.State == "" {
			return html.Div(html.Props{Class: "df-dm-dice", Hidden: true})
		}
		label := view.Kind
		if label == "" {
			label = "check"
		}
		return html.Section(html.Props{Class: "df-dm-dice", Role: "status", Aria: map[string]string{"label": "Dice result"}},
			html.P(html.Props{Class: "df-eyebrow"}, ui.Text(label+" roll")),
			html.Div(html.Props{Class: "df-dm-dice-face"}, ui.Text(strconv.Itoa(int(view.D20)))),
			html.P(html.Props{Class: "df-dm-dice-modifier"}, ui.Text(modifierText(view))),
			diceOutcome(view),
		)
	}
}

func modifierText(view DiceView) string {
	text := "d20"
	if view.Modifier >= 0 {
		text += " + " + strconv.Itoa(int(view.Modifier))
	} else {
		text += " - " + strconv.Itoa(int(-view.Modifier))
	}
	if view.DC > 0 {
		text += " vs DC " + strconv.Itoa(int(view.DC))
	}
	if view.VsLabel != "" {
		text += " " + view.VsLabel
	}
	return text
}

func diceOutcome(view DiceView) ui.Node {
	text := view.Outcome
	if view.Damage != nil && view.Damage.Total != 0 {
		text += " · " + strconv.Itoa(int(view.Damage.Total)) + " " + view.Damage.Type
	}
	if view.Crit {
		text = "CRITICAL"
	}
	if text == "" {
		return html.P(html.Props{Class: "df-dm-dice-outcome"}, ui.Text(view.State))
	}
	return html.P(html.Props{Class: "df-dm-dice-outcome"}, ui.Text(text))
}

// TimerComponent renders the remaining turn time as a progress bar.
func TimerComponent(view TimerView) router.Component {
	return func(_ router.Attrs) *router.Element {
		props := html.Props{Class: "df-dm-timer", Role: "timer", Aria: map[string]string{"label": timerLabel(view)}}
		return html.Section(props,
			html.Div(html.Props{Class: "df-dm-timer-label"}, ui.Text(timerLabel(view))),
			html.Progress(html.Props{Class: "df-dm-timer-bar", Raw: map[string]any{
				"max":   strconv.FormatInt(view.TotalMS, 10),
				"value": strconv.FormatInt(maxTimerMS(view.RemainingMS), 10),
			}}),
		)
	}
}

func timerLabel(view TimerView) string {
	if view.Frozen {
		return "Turn timer paused"
	}
	return "Turn timer: " + strconv.FormatInt(maxTimerMS(view.RemainingMS), 10) + " milliseconds remaining"
}

func maxTimerMS(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
