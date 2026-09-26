//go:build js && wasm

package main

import (
	"context"
	"syscall/js"

	"github.com/monstercameron/GoWebComponents/v6/router"
)

func main() {
	parseRouter := router.NewHistoryRouter(router.RouterOptions{DefaultRoute: string(RouteDM)})
	registerRoutes(parseRouter, newBootClient())
	parseRouter.Mount("#app")
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
