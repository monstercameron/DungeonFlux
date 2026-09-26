//go:build js && wasm

package main

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/router"
)

func main() {
	parseRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: string(RouteDM)})
	client := newBootClient()
	if client == nil {
		registerBootErrorRoutes(parseRouter)
	} else {
		registerRoutes(parseRouter, client)
	}
	parseRouter.Mount("#app")
	select {}
}

func registerBootErrorRoutes(parseRouter *router.Router) {
	locale := NewLocaleModel(BrowserLocales())
	errorRoute := unavailable(locale.T("phone.client_unavail", nil))
	parseRouter.Register(string(RouteDM), errorRoute)
	parseRouter.Register(string(RoutePhone), errorRoute)
	parseRouter.Register(string(RouteHost), errorRoute)
}

func newBootClient() *Client {
	location := js.Global().Get("location")
	client, err := NewClient(context.Background(), location.Get("origin").String()+"/grpc")
	if err != nil {
		return nil
	}
	reporter := NewClientReporter(client, "", "shell")
	reporter.Start(context.Background())
	reporter.SetActive(true)
	return client
}
