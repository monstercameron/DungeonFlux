//go:build js && wasm

package phone

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// journalScreen renders the accumulated story beats for the Journal tab.
func journalScreen(log *JournalLog, locale string) ui.Node {
	theme := DefaultPhoneTheme()
	entries := log.Entries()
	children := []ui.Node{
		html.H1(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "26px"}}, html.Text(T(locale, "journal.title", nil))),
		html.P(html.Props{Style: map[string]string{"margin": "2px 0 4px", "color": theme.Muted, "font-family": theme.Sans, "font-size": "13px"}}, html.Text(T(locale, "journal.subtitle", nil))),
	}
	if len(entries) == 0 {
		children = append(children, journalEmpty(locale, theme))
	} else {
		for i := len(entries) - 1; i >= 0; i-- {
			children = append(children, journalRow(entries[i], locale, theme))
		}
	}
	return html.Main(html.Props{Class: "df-phone-journal", Role: "main", Style: map[string]string{"display": "grid", "gap": "12px", "align-content": "start"}}, children...)
}

func journalEmpty(locale string, theme PhoneTheme) ui.Node {
	return html.Section(html.Props{Style: map[string]string{
		"padding": "16px", "border": "1px dashed rgba(168,159,140,.42)", "border-radius": theme.BorderRadius, "color": theme.Muted,
	}}, html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": theme.Serif, "font-size": "16px", "font-style": "italic"}}, html.Text(T(locale, "journal.empty", nil))))
}

func journalRow(entry JournalEntry, locale string, theme PhoneTheme) ui.Node {
	speaker := entry.Speaker
	if speaker == "" {
		speaker = T(locale, "journal.dm_speaker", nil)
	}
	return html.Section(html.Props{Class: "df-phone-journal-entry", Style: map[string]string{
		"display": "grid", "gap": "4px", "padding": "11px 13px", "border-left": "3px solid " + theme.Gold,
		"border-radius": "0 8px 8px 0", "background": "rgba(23,26,35,.9)",
	}}, html.Strong(html.Props{Style: map[string]string{"color": theme.GoldBright, "font-family": theme.Serif, "font-size": "14px"}}, html.Text(speaker)),
		html.P(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "16px", "font-style": "italic", "line-height": "1.32"}}, html.Text(entry.Text)))
}
