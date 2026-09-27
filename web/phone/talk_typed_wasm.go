//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type talkTypedProps struct {
	model    *TypedInputModel
	snapshot TypedInputSnapshot
}

// talkTypedInput gives the compact input a form, pending state and visible retry.
func talkTypedInput(props talkTypedProps) ui.Node {
	model := props.model
	refresh := ui.UseState(0)
	snapshot := model.Snapshot()
	change := ui.UseEvent(func(event ui.InputEvent) {
		if model.SetText(event.GetValue()) == nil {
			refresh.Set(refresh.Get() + 1)
		}
	})
	send := ui.UseEvent(func(event ui.FormEvent) {
		event.PreventDefault()
		if !model.Snapshot().CanSubmit {
			return
		}
		pending := model.Submit(context.Background())
		refresh.Set(refresh.Get() + 1)
		go func() {
			model.ApplySay(<-pending)
			refresh.Set(refresh.Get() + 1)
		}()
	})
	theme := DefaultPhoneTheme()
	return html.Form(html.Props{OnSubmit: send, Style: map[string]string{"min-width": "0", "flex": "1 1 auto"}},
		html.Div(html.Props{Style: map[string]string{"display": "flex", "align-items": "center", "gap": "4px", "padding": "0 6px 0 12px", "border": "1px solid rgba(168,159,140,.55)", "border-radius": "24px", "background": "rgba(23,26,35,.94)"}},
			html.Input(html.Props{ID: "talk-message", Class: "df-phone-talk-input-field", Value: snapshot.Text, Placeholder: TypedHint(snapshot.Locale), OnInput: change, Disabled: snapshot.Sending, MaxLength: typedInputLimit, Aria: map[string]string{"label": "Type your response", "describedby": "talk-message-status"}, Style: map[string]string{"min-width": "0", "flex": "1 1 auto", "height": "46px", "border": "0", "background": "transparent", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px"}}),
			html.Button(html.Props{Type: "submit", Disabled: !snapshot.CanSubmit, Aria: map[string]string{"label": T(snapshot.Locale, "phone.talk.send", nil)}, Style: typedSendStyle(snapshot.CanSubmit)}, html.Text(T(snapshot.Locale, "phone.talk.send_glyph", nil))),
		),
		html.P(html.Props{ID: "talk-message-status", Role: typedFeedbackRole(snapshot), Style: typedFeedbackStyle(snapshot)}, html.Text(errorOrStatus(snapshot))),
	)
}

func typedSendStyle(enabled bool) map[string]string {
	style := map[string]string{"width": "42px", "height": "42px", "flex": "0 0 42px", "border": "0", "border-radius": "50%", "background": "transparent", "color": DefaultPhoneTheme().GoldBright, "font-size": "19px", "touch-action": "manipulation", "cursor": "pointer"}
	if !enabled {
		style["opacity"], style["cursor"] = ".4", "default"
	}
	return style
}

func typedFeedbackStyle(snapshot TypedInputSnapshot) map[string]string {
	if snapshot.Error != "" {
		return talkErrorStyle()
	}
	return map[string]string{"margin": "4px 10px 0", "color": DefaultPhoneTheme().Muted, "font-size": "12px", "line-height": "1.4", "min-height": "1.4em"}
}
