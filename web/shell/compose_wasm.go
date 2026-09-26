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
	parseRouter.GoRegisterRoute(string(RouteDM), dm.LobbyComponent(dm.NewLobbyModel("DEMO", "")))
	if client == nil {
		parseRouter.GoRegisterRoute(string(RoutePhone), unavailable("Player client unavailable"))
	} else {
		parseRouter.GoRegisterRoute(string(RoutePhone), JoinScreen(client))
	}
	parseRouter.GoRegisterRoute(string(RouteHost), host.Mount("/grpc"))
	RegisterAboutRoute(parseRouter)
	parseRouter.GoRegisterRoute("/", unavailable("Redirecting to DungeonFlux"), router.Options{Redirect: string(RouteDM)})
}

func unavailable(message string) router.Component {
	return func(router.Attrs) *router.Element {
		return html.Main(html.Props{Class: "df-shell-placeholder"}, html.H1(html.Props{}, ui.Text("DungeonFlux")), html.P(html.Props{}, ui.Text(message)))
	}
}
