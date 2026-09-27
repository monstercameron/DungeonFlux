//go:build js && wasm

package main

import (
	"context"
	"errors"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/router"
)

func main() {
	rememberBootQuery()
	parseRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: string(RouteDM)})
	// The router wraps every re-render in a view transition, and Watch updates
	// arrive faster than one completes, so each skipped transition logs an
	// unhandled "Transition was skipped" rejection. Screens animate themselves.
	parseRouter.SetViewTransitions(false)
	client, err := newBootClient()
	if err != nil {
		reportBootError(err)
		registerBootErrorRoutes(parseRouter, err)
	} else {
		registerRoutes(parseRouter, client)
		installResyncTriggers(client)
	}
	parseRouter.Mount("#app")
	installBrowserAssets(context.Background(), client)
	if err != nil {
		revealNow()
	} else {
		revealWhenReady(RouteForPath(js.Global().Get("location").Get("pathname").String()))
	}
	select {}
}

func registerBootErrorRoutes(parseRouter *router.Router, cause error) {
	locale := NewLocaleModel(BrowserLocales())
	message := locale.T("phone.client_unavail", nil)
	if cause != nil {
		message += ": " + cause.Error()
	}
	errorRoute := unavailable(message)
	parseRouter.Register(string(RouteDM), errorRoute)
	parseRouter.Register(string(RoutePhone), errorRoute)
	parseRouter.Register(string(RouteHost), errorRoute)
}

func newBootClient() (*Client, error) {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return nil, errors.New("shell boot: browser location is unavailable")
	}
	client, err := NewClient(context.Background(), location.Get("origin").String()+"/grpc")
	if err != nil {
		return nil, err
	}
	reporter := NewClientReporter(client, "", "shell")
	reporter.Start(context.Background())
	reporter.SetActive(true)
	return client, nil
}

func reportBootError(err error) {
	if err == nil {
		return
	}
	console := js.Global().Get("console")
	if console.Truthy() {
		console.Call("error", "DungeonFlux shell client unavailable: "+err.Error())
	}
}
