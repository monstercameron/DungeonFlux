//go:build js && wasm

package phone

import (
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// PhoneFrame renders the shared header, status, content region, and bottom
// action affordance around a phone screen. Callers can use it when composing
// a screen in the shell without duplicating accessibility or theme tokens.
func PhoneFrame(model FrameModel, content ui.Node, action ui.Node) router.Component {
	return func(_ router.Attrs) *router.Element {
		theme := DefaultPhoneTheme()
		status := ConnectionLabel(model.Locale, model.Connection)
		statusClass := "df-phone-connection df-phone-connection-" + string(model.Connection)
		if model.Connection == "" {
			statusClass = "df-phone-connection df-phone-connection-offline"
		}
		header := phoneFrameHeader(model, statusClass, status)
		body := html.Section(html.Props{Class: "df-phone-frame-content", Role: "region", Aria: map[string]string{"label": model.Title}, Style: map[string]string{"flex": "1 1 auto", "min-height": "0", "overflow": "auto", "padding": "12px 14px 18px"}}, content)
		footer := html.Footer(html.Props{Class: "df-phone-action", Style: map[string]string{"min-height": theme.TouchTarget}}, action, phoneTabBar(model))
		return html.Main(html.Props{Class: "df-phone df-phone-frame", Style: phoneFrameStyle(theme)}, phoneFinishStyle(), header, body, footer)
	}
}

func phoneFrameStyle(theme PhoneTheme) map[string]string {
	return map[string]string{
		"width": "100%", "max-width": theme.PhoneMax, "height": "100dvh", "min-height": "100svh", "box-sizing": "border-box", "margin": "0 auto",
		"overflow": "hidden", "background": theme.InkDeep, "color": theme.Parchment, "font-family": theme.Sans, "display": "flex", "flex-direction": "column",
		"padding": "env(safe-area-inset-top) 0 env(safe-area-inset-bottom)", "transition": "opacity " + theme.Transition,
	}
}

func phoneFrameHeader(model FrameModel, statusClass, status string) ui.Node {
	theme := DefaultPhoneTheme()
	location := model.Location
	if strings.TrimSpace(location) == "" {
		location = model.Title
	}
	act, scene := model.Act, model.Scene
	if act == "" {
		act = "Act I"
	}
	if scene == "" {
		scene = "Scene 1"
	}
	return html.Header(html.Props{Class: "df-phone-header", Style: map[string]string{"height": theme.HeaderHeight, "box-sizing": "border-box", "flex": "0 0 " + theme.HeaderHeight, "display": "flex", "align-items": "center", "justify-content": "space-between", "gap": "10px", "padding": "8px 16px", "border-bottom": "1px solid rgba(217,164,65,.55)", "background": "linear-gradient(180deg, rgba(24,28,37,.98), rgba(11,15,22,.98))"}}, html.Div(html.Props{Class: "df-phone-brand", Style: map[string]string{"min-width": "0"}}, html.Div(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "26px", "font-weight": "600", "letter-spacing": "-.035em", "line-height": "1"}}, html.Span(html.Props{Style: map[string]string{"color": theme.GoldBright}}, html.Text("Dungeon")), html.Span(html.Props{Style: map[string]string{"color": "#8fc6e8"}}, html.Text("Flux"))), html.Span(html.Props{Class: "df-phone-seat", Style: map[string]string{"display": "none"}}, html.Text(model.DisplayName))), html.Div(html.Props{Style: map[string]string{"min-width": "0", "text-align": "right"}}, html.Div(html.Props{Style: map[string]string{"overflow": "hidden", "color": theme.GoldBright, "font-family": theme.Serif, "font-size": "15px", "text-overflow": "ellipsis", "white-space": "nowrap"}}, html.Text(location)), html.Div(html.Props{Style: map[string]string{"color": theme.Muted, "font-family": theme.Serif, "font-size": "12px"}}, html.Text(act+" · "+scene)), html.Span(html.Props{Class: statusClass, Role: "status", Aria: map[string]string{"live": "polite"}, Style: map[string]string{"display": "none"}}, html.Text(status))))
}

func phoneTabBar(model FrameModel) ui.Node {
	theme := DefaultPhoneTheme()
	tabs := FrameTabs(model.ActiveTab, model.Mode)
	items := make([]ui.Node, 0, len(tabs))
	for _, tab := range tabs {
		style := map[string]string{"min-width": "0", "min-height": theme.TouchTarget, "position": "relative", "display": "grid", "place-items": "center", "gap": "2px", "padding": "6px 2px 4px", "border": "0", "background": "transparent", "color": theme.Muted, "font-family": theme.Sans, "font-size": "11px", "line-height": "1", "touch-action": "manipulation"}
		if tab.Active {
			style["color"] = theme.GoldBright
		}
		if tab.Raised {
			style["margin-top"] = "-20px"
			style["min-height"] = "68px"
			style["border"] = "1px solid " + theme.GoldBright
			style["border-radius"] = "8px"
			style["background"] = "linear-gradient(180deg, rgba(57,43,24,.98), rgba(22,20,18,.98))"
			style["box-shadow"] = "0 0 18px rgba(217,164,65,.24), inset 0 0 12px rgba(217,164,65,.12)"
		}
		items = append(items, html.Button(html.Props{Type: "button", Class: "df-phone-tab df-phone-tab-" + string(tab.ID), Aria: map[string]string{"current": currentTabValue(tab.Active), "label": tab.Label}, Style: style}, html.Span(html.Props{Style: map[string]string{"font-family": theme.Serif, "font-size": "24px", "line-height": "1"}}, html.Text(tab.Icon)), html.Span(html.Props{}, html.Text(tab.Label))))
	}
	return html.Nav(html.Props{Class: "df-phone-tabs", Aria: map[string]string{"label": "Phone navigation"}, Style: map[string]string{"height": theme.TabBarHeight, "box-sizing": "border-box", "flex": "0 0 " + theme.TabBarHeight, "display": "grid", "grid-template-columns": "repeat(5, minmax(0, 1fr))", "align-items": "end", "gap": "4px", "padding": "8px 10px calc(6px + env(safe-area-inset-bottom))", "border-top": "1px solid rgba(168,159,140,.26)", "background": "rgba(11,15,22,.98)"}}, items...)
}

func currentTabValue(active bool) string {
	if active {
		return "page"
	}
	return "false"
}
