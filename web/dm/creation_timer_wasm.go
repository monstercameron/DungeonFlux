//go:build js && wasm

package dm

import (
	"strconv"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type creationTimerAnchor struct {
	remainingMS int64
	totalMS     int64
	frozen      bool
	atMS        float64
}

func creationTimerAnchorFor(ref ui.Ref[creationTimerAnchor], timer TimerView, now float64) creationTimerAnchor {
	anchor := ref.Get()
	if anchor.remainingMS != timer.RemainingMS || anchor.totalMS != timer.TotalMS || anchor.frozen != timer.Frozen {
		anchor = creationTimerAnchor{remainingMS: timer.RemainingMS, totalMS: timer.TotalMS, frozen: timer.Frozen, atMS: now}
		ref.Set(anchor)
	}
	return anchor
}

func creationTimerSnapshot(timer TimerView, anchor creationTimerAnchor, now float64) TimerView {
	if timer.TotalMS <= 0 || timer.Frozen || timer.RemainingMS <= 0 {
		return timer
	}
	timer.RemainingMS = creationTimerRemainingAt(timer.RemainingMS, anchor.atMS, now)
	return timer
}

func creationTimerNode(timer TimerView) ui.Node {
	if timer.TotalMS <= 0 {
		return nil
	}
	percent := timer.RemainingMS * 100 / timer.TotalMS
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	seconds := (timer.RemainingMS + 999) / 1000
	label := "Time to choose · " + strconv.FormatInt(seconds, 10) + "s"
	if timer.Frozen {
		label = "Creation paused · " + strconv.FormatInt(seconds, 10) + "s"
	}
	return html.Div(html.Props{Class: "df-dm-creation-timer", Role: "timer", Aria: map[string]string{"label": label}, Style: map[string]string{
		"position": "absolute", "right": "42px", "top": "124px", "z-index": "5", "width": "250px", "padding": "12px 16px",
		"border": "1px solid rgba(217,164,65,.78)", "border-radius": "8px", "background": "rgba(8,11,16,.88)", "color": "#e7c27a",
		"font-family": "Cinzel, Georgia, serif", "font-size": "18px", "text-align": "center",
	}},
		html.Div(html.Props{}, html.Text(label)),
		html.Div(html.Props{Role: "progressbar", Aria: map[string]string{"label": label, "valuenow": strconv.FormatInt(percent, 10), "valuemin": "0", "valuemax": "100"}, Style: map[string]string{"height": "5px", "margin-top": "8px", "background": "rgba(255,255,255,.12)", "border-radius": "4px", "overflow": "hidden"}},
			html.Span(html.Props{Style: map[string]string{"display": "block", "height": "100%", "width": strconv.FormatInt(percent, 10) + "%", "background": "#d9a441"}})),
	)
}
