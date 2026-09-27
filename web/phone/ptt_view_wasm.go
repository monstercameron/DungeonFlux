//go:build js && wasm

package phone

import (
	"time"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type talkPTTProps struct {
	model  *PTTModel
	locale string
}

// talkPTTScreen renders the tap-to-talk microphone: one tap starts recording,
// the next sends it. The recording lives on the PTTModel (ToggleControl and
// browserTalk), not in this component, so a remount while recording keeps the
// microphone open. Status and errors are shown on screen.
func talkPTTScreen(props talkPTTProps) ui.Node {
	locale := props.locale
	if locale == "" {
		locale = "en"
	}
	control := props.model.Toggle()
	revision := ui.UseState(0)
	if props.model != nil {
		browserTalkFor(props.model).setRefresh(func() { revision.Set(revision.Get() + 1) })
	}
	tap := ui.UseEvent(func(event ui.Event) {
		event.PreventDefault()
		if control == nil || props.model.opener == nil {
			return
		}
		switch control.Tap(time.Now()) {
		case ToggleStart:
			go beginTalk(props.model, locale)
		case ToggleFinish:
			go endTalk(props.model)
		}
		revision.Set(revision.Get() + 1)
	})
	phase := ToggleIdle
	notice := T(locale, "ptt.unavail", nil)
	if control != nil {
		phase, notice = control.Phase(), control.Notice()
	}
	status, label, aria := talkStatus(locale, phase, notice)
	// Fixtures use the same visible lifecycle without opening a microphone.
	snapshot := props.model.ControlSnapshot(locale)
	recording := phase == ToggleRecording || phase == ToggleStarting
	if props.model != nil && props.model.opener == nil && snapshot.State != PTTIdle {
		status, aria = snapshot.StatusText, snapshot.StatusText
		recording = snapshot.State == PTTRecording
		switch snapshot.State {
		case PTTRecording:
			label = "■"
		case PTTStopping, PTTTranscribing:
			label = "…"
		case PTTFailed:
			label = "!"
		}
	}
	if phase == ToggleIdle && notice != "" {
		label = "!"
	}

	return html.Div(html.Props{Class: "df-phone-talk-ptt", Style: map[string]string{"display": "flex", "flex-direction": "column", "align-items": "center", "gap": "7px", "flex": "0 0 76px"}},
		html.Button(html.Props{Type: "button", OnClick: tap, Aria: map[string]string{"label": aria}, Style: talkMicStyle(recording)}, html.Text(label)),
		html.Span(html.Props{Role: "status", Style: map[string]string{"font-size": "11px", "line-height": "1.3", "text-align": "center", "color": "#d7cdbb", "overflow-wrap": "anywhere"}}, html.Text(status)),
	)
}

func talkMicStyle(recording bool) map[string]string {
	theme := DefaultPhoneTheme()
	background := "rgba(29,33,44,.96)"
	color := theme.GoldBright
	if recording {
		background = "rgba(179,55,47,.85)"
		color = theme.Parchment
	}
	return map[string]string{"width": "50px", "height": "50px", "flex": "0 0 50px", "border": "1px solid " + theme.GoldBright, "border-radius": "50%", "background": background, "color": color, "font-size": "18px", "box-shadow": "0 0 14px rgba(217,164,65,.18)", "touch-action": "manipulation"}
}
