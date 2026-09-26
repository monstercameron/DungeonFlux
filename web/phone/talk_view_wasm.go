//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// TalkScreen renders the NPC portrait, legal conversation moves, and input
// controls inside the shared phone frame.
func TalkScreen(props phoneViewProps, locale string) router.Component {
	return func(_ router.Attrs) *router.Element {
		refresh := ui.UseState(0)
		moves := props.moves.Snapshot()
		talk := NewTalkSnapshot(moves.StatusText)
		rows := TalkMoveRows(moves.Moves)
		children := make([]ui.Node, 0, len(rows))
		for _, row := range rows {
			item := row
			tap := ui.UseEvent(func() {
				go func() {
					props.moves.ApplyAct(<-props.moves.Tap(context.Background(), item.ID))
					refresh.Set(refresh.Get() + 1)
				}()
			})
			children = append(children, ChoiceRow(item, tap))
		}
		if len(children) == 0 {
			children = append(children, html.P(html.Props{Style: talkMutedTextStyle()}, html.Text("Marra waits for your answer.")))
		}
		if moves.Error != "" {
			children = append(children, html.P(html.Props{Role: "alert", Style: talkErrorStyle()}, html.Text(moves.Error)))
		}
		portrait := ArtURL(talk.PortraitArt)
		return html.Main(html.Props{Class: "df-phone df-phone-conversation df-phone-talk", Role: "main", Style: talkPageStyle()},
			PortraitHero(portrait, talk.Speaker, talk.Role, talk.Quote),
			html.Section(html.Props{Class: "df-phone-talk-choices", Aria: map[string]string{"label": "Conversation choices"}, Style: talkChoicesStyle()}, children...),
			talkInputBar(props, locale),
		)
	}
}

func talkPageStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"display": "flex", "flex-direction": "column", "gap": "12px", "min-height": "100%", "color": theme.Parchment, "font-family": theme.Sans}
}

func talkChoicesStyle() map[string]string {
	return map[string]string{"display": "flex", "flex-direction": "column", "gap": "8px", "padding": "0 2px"}
}

func talkMutedTextStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"margin": "0", "padding": "14px", "border": "1px solid rgba(168,159,140,.3)", "border-radius": theme.BorderRadius, "color": theme.Muted, "font-family": theme.Serif, "font-size": "17px", "font-style": "italic"}
}

func talkErrorStyle() map[string]string {
	theme := DefaultPhoneTheme()
	return map[string]string{"margin": "0", "padding": "10px 12px", "border": "1px solid " + theme.Blood, "border-radius": "8px", "background": "rgba(179,55,47,.18)", "color": "#ffd8d2", "font-size": "13px"}
}

func talkInputBar(props phoneViewProps, locale string) ui.Node {
	theme := DefaultPhoneTheme()
	return html.Div(html.Props{Class: "df-phone-talk-input", Style: map[string]string{"display": "flex", "align-items": "flex-end", "gap": "8px", "padding": "8px 0 2px", "border-top": "1px solid rgba(217,164,65,.25)"}},
		ui.CreateElement(talkPTTScreen, talkPTTProps{model: props.ptt, locale: locale}),
		html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1 1 auto", "display": "flex", "align-items": "center", "gap": "6px", "min-height": theme.TouchTarget, "padding": "0 6px 0 12px", "border": "1px solid rgba(168,159,140,.55)", "border-radius": "24px", "background": "rgba(23,26,35,.94)"}}, html.CreateElement(talkTypedInput, props.typed)),
	)
}

// talkTypedInput keeps the text fallback compact enough to share a row with
// the round microphone control.
func talkTypedInput(model *TypedInputModel) ui.Node {
	refresh := ui.UseState(0)
	snapshot := model.Snapshot()
	change := ui.UseEvent(func(event ui.InputEvent) {
		if model.SetText(event.GetValue()) == nil {
			refresh.Set(refresh.Get() + 1)
		}
	})
	send := ui.UseEvent(func() {
		go func() {
			model.ApplySay(<-model.Submit(context.Background()))
			refresh.Set(refresh.Get() + 1)
		}()
	})
	placeholder := TypedHint(snapshot.Locale)
	if placeholder == "" {
		placeholder = "Or speak your response…"
	}
	return html.Div(html.Props{Style: map[string]string{"min-width": "0", "flex": "1 1 auto", "display": "flex", "align-items": "center", "gap": "4px"}},
		html.Input(html.Props{ID: "talk-message", Class: "df-phone-talk-input-field", Value: snapshot.Text, Placeholder: placeholder, OnInput: change, Disabled: snapshot.Sending, MaxLength: typedInputLimit, Aria: map[string]string{"label": "Type your response"}, Style: map[string]string{"min-width": "0", "flex": "1 1 auto", "height": "46px", "border": "0", "outline": "0", "background": "transparent", "color": DefaultPhoneTheme().Parchment, "font-family": DefaultPhoneTheme().Serif, "font-size": "16px"}}),
		html.Button(html.Props{Type: "button", OnClick: send, Disabled: !snapshot.CanSubmit, Aria: map[string]string{"label": "Send response"}, Style: map[string]string{"width": "42px", "height": "42px", "flex": "0 0 42px", "border": "0", "border-radius": "50%", "background": "transparent", "color": DefaultPhoneTheme().GoldBright, "font-size": "19px", "touch-action": "manipulation"}}, html.Text("➤")),
	)
}
