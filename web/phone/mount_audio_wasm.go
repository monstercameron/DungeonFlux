//go:build js && wasm

package phone

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

type phoneSeatProvider interface{ PlayerNumber() int32 }

func newPhoneAudio(client PhoneClient, seatToken string) *PhoneAudio {
	audio := NewPhoneAudio()
	service, ok := client.(phoneAudioService)
	if !ok {
		return audio
	}
	seat := int32(0)
	if provider, ok := client.(phoneSeatProvider); ok {
		seat = provider.PlayerNumber()
	}
	audio.ConfigureAudio(service, seatToken, seat)
	return audio
}

func audioControls(audio *PhoneAudio, locale string) ui.Node {
	if audio == nil || audio.service == nil {
		return nil
	}
	muted := ui.UseState(false)
	unlock := ui.UseEvent(func() { _ = audio.UnlockAudio(context.Background()) })
	toggle := ui.UseEvent(func() {
		next := !muted.Get()
		muted.Set(next)
		audio.SetMuted(next)
	})
	label := "Enable sound"
	if locale == "es" {
		label = "Activar sonido"
	}
	muteLabel := "Mute"
	if muted.Get() {
		muteLabel = "Unmute"
	}
	return html.Div(html.Props{Class: "df-phone-audio-controls", Style: map[string]string{"display": "flex", "gap": "6px", "justify-content": "center", "padding": "4px"}},
		html.Button(html.Props{Type: "button", OnClick: unlock}, html.Text(label)),
		html.Button(html.Props{Type: "button", OnClick: toggle}, html.Text(muteLabel)))
}
