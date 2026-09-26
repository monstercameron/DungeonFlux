//go:build js && wasm

package main

import (
	"github.com/monstercameron/DungeonFlux/web/dm"
	"github.com/monstercameron/DungeonFlux/web/host"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func registerRoutes(parseRouter *router.Router, client *Client) {
	locale := NewLocaleModel(BrowserLocales())
	parseRouter.Register(string(RouteDM), PreviewComponent(dm.Mount("/grpc"), dmPreviews()))
	if client == nil {
		parseRouter.Register(string(RoutePhone), unavailable(locale.T("phone.client_unavail", nil)))
	} else {
		parseRouter.Register(string(RoutePhone), PreviewComponent(JoinScreen(client), phonePreviews()))
	}
	parseRouter.Register(string(RouteHost), host.Mount("/grpc"))
	RegisterAboutRoute(parseRouter)
	RegisterPreviewRoute(parseRouter, NewPreviewCatalog(dmPreviews(), phonePreviews()))
	parseRouter.Register("/", unavailable(locale.T("shell.redirect", nil)), router.Options{Redirect: string(RouteDM)})
}

func unavailable(message string) router.Component {
	locale := NewLocaleModel(BrowserLocales())
	return func(router.Attrs) *router.Element {
		return html.Main(html.Props{Class: "df-shell-placeholder"}, html.H1(html.Props{}, ui.Text(locale.T("shell.brand", nil))), html.P(html.Props{}, ui.Text(message)))
	}
}
