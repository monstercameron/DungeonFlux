//go:build js && wasm

package phone

import (
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// menuSeatTokenPrefix must be kept in sync with web/shell's
// phoneSeatStoragePrefix (web/shell/seat_store.go); the phone package
// cannot import package main, so Leave the table clears it by prefix.
const menuSeatTokenPrefix = "dungeonflux.phone.seat_token"

// menuScreen renders language, sound, help, and leave-table controls.
func menuScreen(audio *PhoneAudio, locale string) ui.Node {
	theme := DefaultPhoneTheme()
	muted := ui.UseState(false)
	confirmingLeave := ui.UseState(false)

	setEnglish := ui.UseEvent(func() { SetLocaleOverride("en") })
	setSpanish := ui.UseEvent(func() { SetLocaleOverride("es") })
	toggleSound := ui.UseEvent(func() {
		next := !muted.Get()
		muted.Set(next)
		if audio != nil {
			audio.SetMuted(next)
		}
	})
	askLeave := ui.UseEvent(func() { confirmingLeave.Set(true) })
	cancelLeave := ui.UseEvent(func() { confirmingLeave.Set(false) })
	confirmLeave := ui.UseEvent(func() { leaveTable() })

	return html.Main(html.Props{Class: "df-phone-menu", Role: "main", Style: map[string]string{"display": "grid", "gap": "16px", "align-content": "start"}},
		html.H1(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "26px"}}, html.Text(T(locale, "menu.title", nil))),
		menuSection(theme, T(locale, "menu.language", nil), menuChoiceRow(theme,
			menuChoiceButton(theme, T(locale, "menu.language_en", nil), setEnglish, locale == "en"),
			menuChoiceButton(theme, T(locale, "menu.language_es", nil), setSpanish, locale == "es"),
		)),
		menuSection(theme, T(locale, "menu.sound", nil), SecondaryButton(soundLabel(locale, muted.Get()), toggleSound, false)),
		menuSection(theme, T(locale, "menu.help", nil), html.P(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px", "line-height": "1.35"}}, html.Text(T(locale, "menu.help_body", nil)))),
		menuLeaveSection(theme, locale, confirmingLeave.Get(), askLeave, cancelLeave, confirmLeave),
	)
}

func soundLabel(locale string, muted bool) string {
	if muted {
		return T(locale, "menu.sound_off", nil)
	}
	return T(locale, "menu.sound_on", nil)
}

func menuSection(theme PhoneTheme, title string, content ui.Node) ui.Node {
	return html.Section(html.Props{Style: map[string]string{"display": "grid", "gap": "8px"}},
		html.Strong(html.Props{Style: map[string]string{"color": theme.GoldBright, "font-family": theme.Sans, "font-size": "12px", "letter-spacing": ".05em", "text-transform": "uppercase"}}, html.Text(title)),
		content,
	)
}

func menuChoiceRow(theme PhoneTheme, buttons ...ui.Node) ui.Node {
	return html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "repeat(2, minmax(0, 1fr))", "gap": "8px"}}, buttons...)
}

func menuChoiceButton(theme PhoneTheme, label string, tap ui.Handler, active bool) ui.Node {
	style := map[string]string{
		"min-height": "48px", "padding": "10px 14px", "border-radius": "8px", "font-family": theme.Serif, "font-size": "16px",
		"touch-action": "manipulation", "border": "1px solid rgba(168,159,140,.5)", "background": "rgba(15,17,23,.82)", "color": theme.Parchment,
	}
	if active {
		style["border"] = "1px solid " + theme.GoldBright
		style["background"] = "linear-gradient(180deg, rgba(64,48,25,.92), rgba(25,22,19,.98))"
		style["box-shadow"] = "inset 0 0 12px rgba(217,164,65,.16)"
	}
	return html.Button(html.Props{Type: "button", OnClick: tap, Aria: map[string]string{"pressed": boolAttr(active)}, Style: style}, html.Text(label))
}

func boolAttr(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func menuLeaveSection(theme PhoneTheme, locale string, confirming bool, ask, cancel, confirm ui.Handler) ui.Node {
	if !confirming {
		return menuSection(theme, "", dangerButton(theme, T(locale, "menu.leave", nil), ask))
	}
	return menuSection(theme, "", html.Div(html.Props{Class: "df-phone-menu-confirm", Style: map[string]string{
		"display": "grid", "gap": "10px", "padding": "14px", "border": "1px solid " + theme.Blood, "border-radius": theme.BorderRadius, "background": "rgba(24,12,12,.5)",
	}},
		html.Strong(html.Props{Style: map[string]string{"color": theme.Parchment, "font-family": theme.Serif, "font-size": "18px"}}, html.Text(T(locale, "menu.leave_confirm_title", nil))),
		html.P(html.Props{Style: map[string]string{"margin": "0", "color": theme.Muted, "font-family": theme.Sans, "font-size": "13px"}}, html.Text(T(locale, "menu.leave_confirm_body", nil))),
		html.Div(html.Props{Style: map[string]string{"display": "grid", "grid-template-columns": "1fr 1fr", "gap": "8px"}},
			SecondaryButton(T(locale, "menu.leave_confirm_cancel", nil), cancel, false),
			dangerButton(theme, T(locale, "menu.leave_confirm_yes", nil), confirm),
		),
	))
}

func dangerButton(theme PhoneTheme, label string, tap ui.Handler) ui.Node {
	return html.Button(html.Props{Type: "button", OnClick: tap, Style: map[string]string{
		"width": "100%", "min-height": "52px", "padding": "9px 18px", "border": "1px solid " + theme.Blood, "border-radius": "7px",
		"background": "rgba(30,10,10,.7)", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "17px", "touch-action": "manipulation",
	}}, html.Text(label))
}

// leaveTable clears every saved phone seat token on this device and returns
// to the join screen. There is no server-side "leave" verb in the demo
// engine (plan §0.5); dropping the local seat token and reloading /p is the
// client-side equivalent, matching what a stale-token reload already does
// (WEB-014).
func leaveTable() {
	storage := js.Global().Get("localStorage")
	if storage.Truthy() {
		length := storage.Get("length").Int()
		var toRemove []string
		for i := 0; i < length; i++ {
			key := storage.Call("key", i)
			if key.Truthy() && len(key.String()) >= len(menuSeatTokenPrefix) && key.String()[:len(menuSeatTokenPrefix)] == menuSeatTokenPrefix {
				toRemove = append(toRemove, key.String())
			}
		}
		for _, key := range toRemove {
			storage.Call("removeItem", key)
		}
	}
	location := js.Global().Get("location")
	if location.Truthy() {
		location.Set("href", "/p")
	}
}
