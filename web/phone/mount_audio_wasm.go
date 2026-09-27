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

// audioControls renders the compact sound pill shown above the tab bar on
// every screen. The first tap anywhere already unlocks Web Audio (see
// installTapListener in audio_wasm.go), so this is a single styled
// mute/unmute toggle rather than a separate, easy-to-miss "Enable sound"
// step; its handler also unlocks audio, which is a harmless no-op once a
// tap already has. The same setting lives on the Menu tab too.
func audioControls(audio *PhoneAudio, locale string) ui.Node {
	if audio == nil || audio.service == nil {
		return nil
	}
	theme := DefaultPhoneTheme()
	muted := ui.UseState(false)
	toggle := ui.UseEvent(func() {
		_ = audio.UnlockAudio(context.Background())
		next := !muted.Get()
		muted.Set(next)
		audio.SetMuted(next)
	})
	pillStyle := map[string]string{
		"min-height": "36px", "padding": "6px 16px", "border-radius": "999px", "border": "1px solid rgba(217,164,65,.6)",
		"background": "rgba(23,26,35,.9)", "color": theme.GoldBright, "font-family": theme.Sans, "font-size": "12px",
		"font-weight": "600", "letter-spacing": ".02em", "touch-action": "manipulation",
	}
	label := T(locale, "phone.audio.mute", nil)
	if muted.Get() {
		label = T(locale, "phone.audio.unmute", nil)
		pillStyle["color"] = theme.Muted
		pillStyle["border"] = "1px solid rgba(168,159,140,.5)"
	}
	return html.Div(html.Props{Class: "df-phone-audio-controls", Style: map[string]string{"display": "flex", "justify-content": "center", "padding": "4px 0"}},
		html.Button(html.Props{Type: "button", Class: "df-phone-audio-toggle", OnClick: toggle, Aria: map[string]string{"pressed": boolAttr(muted.Get())}, Style: pillStyle}, html.Text(label)))
}
