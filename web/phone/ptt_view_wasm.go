//go:build js && wasm

package phone

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type talkPTTProps struct {
	model  *PTTModel
	locale string
}

// talkPTTScreen renders a single round hold-to-talk control for conversation.
func talkPTTScreen(props talkPTTProps) ui.Node {
	locale := props.locale
	if locale == "" {
		locale = "en"
	}
	status := ui.UseState(PTTReady(locale))
	busy := ui.UseState(false)
	recorder := ui.UseState((*BrowserRecorder)(nil))
	stream := ui.UseState(js.Value{})
	cancelRecording := ui.UseState((context.CancelFunc)(nil))
	ui.UseEffect(func() func() {
		return func() {
			if cancel := cancelRecording.Get(); cancel != nil {
				cancel()
			}
			if current := recorder.Get(); current != nil {
				current.Dispose()
			}
			stopTracks(stream.Get())
		}
	}, props.model)
	start := ui.UseEvent(func() {
		if props.model == nil || props.model.opener == nil {
			return
		}
		if busy.Get() {
			current := recorder.Get()
			if current != nil {
				status.Set(T(locale, "ptt.finishing", nil))
				go finishPTT(current, props.model, locale, stream.Get(), cancelRecording.Get, status.Set, busy.Set)
			}
			return
		}
		busy.Set(true)
		ctx, cancel := context.WithCancel(context.Background())
		cancelRecording.Set(cancel)
		go startPTT(ctx, cancel, props.model, locale, status.Set, busy.Set, recorder.Set, stream.Set)
	})
	snapshot := props.model.ControlSnapshot(locale)
	label, caption := "●", status.Get()
	recording := snapshot.State == PTTRecording || busy.Get()
	if recording {
		label = "■"
	}
	if snapshot.State == PTTTranscribing || snapshot.State == PTTStopping {
		label = "…"
	}
	if snapshot.State == PTTFailed {
		label = "!"
	}
	if snapshot.State != PTTIdle {
		caption = snapshot.StatusText
	}
	return html.Div(html.Props{Class: "df-phone-talk-ptt", Style: map[string]string{"display": "flex", "flex-direction": "column", "align-items": "center", "gap": "7px", "flex": "0 0 76px"}},
		html.Button(html.Props{Type: "button", OnClick: start, Aria: map[string]string{"label": caption}, Style: talkMicStyle(recording)}, html.Text(label)),
		html.Span(html.Props{Role: "status", Style: map[string]string{"font-size": "11px", "line-height": "1.3", "text-align": "center", "color": "#d7cdbb"}}, html.Text(caption)),
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
