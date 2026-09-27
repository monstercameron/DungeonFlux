//go:build js && wasm

package main

import "github.com/monstercameron/DungeonFlux/web/shell/watch"

func installResyncTriggers(client *Client) {
	if client != nil {
		watch.InstallTriggers(client)
	}
}
