//go:build js && wasm

package main

import (
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func main() {
	parseRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: string(RouteDM)})
	parseRouter.GoRegisterRoute(string(RouteDM), placeholder("DungeonFlux DM"))
	parseRouter.GoRegisterRoute(string(RoutePhone), placeholder("DungeonFlux Player"))
	parseRouter.GoRegisterRoute(string(RouteHost), placeholder("DungeonFlux Host"))
	parseRouter.GoRegisterRoute("/", placeholder("DungeonFlux DM"), router.Options{Redirect: string(RouteDM)})
	parseRouter.Mount("#app")
}

func placeholder(title string) router.Component {
	return func(router.Attrs) *router.Element {
		return html.Main(
			html.Props{Class: "df-shell-placeholder"},
			html.H1(html.Props{}, ui.Text(title)),
			html.P(html.Props{}, ui.Text("DungeonFlux is loading this client.")),
		)
	}
}
