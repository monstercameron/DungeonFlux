//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// AboutPage renders the public licensing and attribution notice.
func AboutPage(router.Attrs) *router.Element {
	locale := NewLocaleModel(BrowserLocales())
	return html.Main(html.Props{Class: "df-shell-about"},
		html.H1(html.Props{}, ui.Text(locale.T("about.title", nil))),
		html.Section(html.Props{},
			html.H2(html.Props{}, ui.Text(locale.T("about.rules", nil))),
			html.P(html.Props{}, ui.Text(SRDAttribution)),
			html.P(html.Props{}, ui.Text(DrownedThrallNotice)),
		),
	)
}

// RegisterAboutRoute adds the attribution page to a shell router.
func RegisterAboutRoute(parseRouter *router.Router) {
	if parseRouter == nil {
		return
	}
	parseRouter.Register(string(RouteAbout), AboutPage)
}
