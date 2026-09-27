//go:build js && wasm

package phone

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// mapScreen renders the party's current scene, and, in combat, a link back
// to the Play tab's top-down battle map (COMBAT-MOVE owns the map itself;
// this tab only orients the player and hands them back to it).
func mapScreen(view SeatView, locale string, openCombatMap ui.Handler) ui.Node {
	theme := DefaultPhoneTheme()
	imageURL := strings.TrimSpace(view.Phone.GetSceneImageUrl())
	inCombat := view.Phone.GetCombat() != nil
	children := []ui.Node{
		html.H1(html.Props{Style: map[string]string{"margin": "0", "color": theme.Parchment, "font-family": theme.Serif, "font-size": "26px"}}, html.Text(T(locale, "map.title", nil))),
		html.P(html.Props{Style: map[string]string{"margin": "2px 0 4px", "color": theme.Muted, "font-family": theme.Sans, "font-size": "13px"}}, html.Text(T(locale, "map.subtitle", nil))),
	}
	if imageURL != "" {
		children = append(children, html.Section(html.Props{Style: map[string]string{
			"position": "relative", "min-height": "220px", "overflow": "hidden", "border-radius": theme.BorderRadius,
			"border": "1px solid rgba(217,164,65,.5)", "background": "linear-gradient(180deg, transparent 55%, rgba(7,10,15,.92) 100%), url(\"" + imageURL + "\") center / cover",
		}}, html.Div(html.Props{Style: map[string]string{"position": "absolute", "left": "14px", "right": "14px", "bottom": "12px"}},
			html.Strong(html.Props{Style: map[string]string{"color": theme.GoldBright, "font-family": theme.Serif, "font-size": "18px"}}, html.Text(T(locale, "map.location", nil))))))
	} else {
		children = append(children, html.Section(html.Props{Style: map[string]string{
			"padding": "16px", "border": "1px dashed rgba(168,159,140,.42)", "border-radius": theme.BorderRadius, "color": theme.Muted,
		}}, html.P(html.Props{Style: map[string]string{"margin": "0", "font-family": theme.Serif, "font-size": "16px", "font-style": "italic"}}, html.Text(T(locale, "map.empty", nil)))))
	}
	if inCombat {
		children = append(children, PrimaryButton(T(locale, "map.combat_link", nil), openCombatMap, false))
	}
	return html.Main(html.Props{Class: "df-phone-map", Role: "main", Style: map[string]string{"display": "grid", "gap": "12px", "align-content": "start"}}, children...)
}
