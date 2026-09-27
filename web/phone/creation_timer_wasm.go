//go:build js && wasm

package phone

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

func creationTimerAnchorFor(ref ui.Ref[creationTimerAnchor], snapshot CreationSnapshot, now float64) creationTimerAnchor {
	anchor := ref.Get()
	if anchor.totalMS != snapshot.TimerTotalMS || anchor.remainingMS != snapshot.TimerRemainingMS || anchor.frozen != snapshot.TimerFrozen {
		anchor = creationTimerAnchor{remainingMS: snapshot.TimerRemainingMS, totalMS: snapshot.TimerTotalMS, frozen: snapshot.TimerFrozen, atMS: now}
		ref.Set(anchor)
	}
	return anchor
}

func creationTimerSnapshot(snapshot CreationSnapshot, anchor creationTimerAnchor, now float64) CreationSnapshot {
	if snapshot.TimerTotalMS <= 0 || snapshot.TimerFrozen || now <= anchor.atMS {
		return snapshot
	}
	snapshot.TimerRemainingMS = creationTimerRemainingAt(snapshot.TimerRemainingMS, anchor.atMS, now)
	return snapshot
}

func creationTimerNode(snapshot CreationSnapshot) ui.Node {
	if snapshot.TimerTotalMS <= 0 {
		return nil
	}
	percent := int64(0)
	if snapshot.TimerTotalMS > 0 {
		percent = snapshot.TimerRemainingMS * 100 / snapshot.TimerTotalMS
	}
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	label := creationTimerLabel(snapshot.Locale, snapshot.TimerRemainingMS, snapshot.TimerFrozen)
	return html.Div(html.Props{Class: "df-phone-create-timer", Role: "timer", Aria: map[string]string{"label": label}, Style: map[string]string{
		"display": "grid", "gap": "5px", "padding": "8px 10px", "border": "1px solid rgba(217,164,65,.42)",
		"border-radius": "9px", "background": "rgba(17,21,29,.84)", "color": "#e7c27a", "font-size": "12px",
	}},
		html.Div(html.Props{Style: map[string]string{"display": "flex", "justify-content": "space-between", "gap": "8px"}},
			html.Span(html.Props{}, html.Text(label))),
		html.Div(html.Props{Role: "progressbar", Aria: map[string]string{"label": label, "valuenow": strconv.FormatInt(percent, 10), "valuemin": "0", "valuemax": "100"}, Style: map[string]string{"height": "4px", "overflow": "hidden", "border-radius": "4px", "background": "rgba(255,255,255,.12)"}},
			html.Span(html.Props{Style: map[string]string{"display": "block", "width": strconv.FormatInt(percent, 10) + "%", "height": "100%", "background": "#d9a441", "transition": "width .25s linear"}})),
	)
}
