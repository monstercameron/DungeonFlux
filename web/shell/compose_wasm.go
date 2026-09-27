//go:build js && wasm

package main

import (
	"context"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/dm"
	"github.com/monstercameron/DungeonFlux/web/host"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
	"google.golang.org/grpc"
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

// Listen exposes the joined shell connection to the phone audio mount.
func (a phoneClientAdapter) Listen(ctx context.Context, request *dungeonfluxv1.ListenRequest, options ...grpc.CallOption) (grpc.ServerStreamingClient[dungeonfluxv1.AudioMessage], error) {
	if a.client == nil {
		return nil, context.Canceled
	}
	return a.client.Listen(ctx, request, options...)
}

// PlayerNumber supplies the seat filter learned during Join.
func (a phoneClientAdapter) PlayerNumber() int32 {
	if a.client == nil {
		return 0
	}
	return a.client.PlayerNumber()
}

func unavailable(message string) router.Component {
	locale := NewLocaleModel(BrowserLocales())
	return func(router.Attrs) *router.Element {
		return html.Main(html.Props{Class: "df-shell-placeholder"}, html.H1(html.Props{}, ui.Text(locale.T("shell.brand", nil))), html.P(html.Props{}, ui.Text(message)))
	}
}
