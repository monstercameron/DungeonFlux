//go:build js && wasm

package host

import (
	"context"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Mount returns the host control screen for the shared shell router.
func Mount(endpoint string) router.Component {
	return func(_ router.Attrs) *router.Element {
		client, err := newHostClient(endpoint)
		if err != nil {
			return ui.CreateElement(errorView, err.Error())
		}
		return ui.CreateElement(hostView, hostViewProps{Client: client})
	}
}

type hostViewProps struct{ Client *hostClient }

func errorView(message string) ui.Node {
	return html.Main(html.Props{Class: "df-host"}, html.H1(html.Props{}, html.Text("DungeonFlux Host")), html.P(html.Props{}, html.Text(message)))
}

func hostView(parseProps hostViewProps) ui.Node {
	status := ui.UseState("Ready")
	query := router.UseQuery()
	token := query.Get("t")
	if token == "" {
		token = query.Get("token")
	}
	buttons := make([]ui.Node, 0, len(hostActions))
	for _, action := range hostActions {
		buttons = append(buttons, ui.CreateElement(hostActionButton, hostActionButtonProps{Action: action, Click: func() { sendCommand(parseProps.Client, token, action, status.Update) }}))
	}
	return html.Main(html.Props{Class: "df-host"}, html.H1(html.Props{}, html.Text("DungeonFlux Host")), html.P(html.Props{Class: "df-host-status", Aria: map[string]string{"live": "polite"}}, html.Text(status.Get())), html.Div(html.Props{Class: "df-host-actions"}, buttons...))
}

type hostActionButtonProps struct {
	Action hostAction
	Click  func()
}

func hostActionButton(parseProps hostActionButtonProps) ui.Node {
	handleClick := ui.UseEvent(func(parseEvent ui.Event) { parseEvent.PreventDefault(); parseProps.Click() })
	return html.Button(html.Props{Type: "button", Class: "df-host-action", OnClick: handleClick}, html.Text(parseProps.Action.Label))
}

func sendCommand(client *hostClient, token string, action hostAction, update func(func(string) string)) {
	update(func(string) string { return "Sending " + action.Label + "…" })
	go func() {
		ack, err := client.command(context.Background(), commandFor(action, token))
		if err != nil {
			update(func(string) string { return "Error: " + err.Error() })
			return
		}
		message := "Accepted"
		if !ack.GetOk() {
			message = "Rejected: " + ack.GetReason()
		}
		update(func(string) string { return message })
	}()
}
