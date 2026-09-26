package main

// Route identifies one of the clients served by the shared WASM bundle.
type Route string

const (
	// RouteDM is the shared-screen dungeon master client.
	RouteDM Route = "/dm"
	// RoutePhone is the player phone client.
	RoutePhone Route = "/p"
	// RouteHost is the host control client.
	RouteHost Route = "/host"
	// RouteNotFound is used when a path is outside the client surface.
	RouteNotFound Route = ""
)

// RouteForPath maps a browser path to the client that should render it.
// Query strings and trailing slashes do not change the selected client.
func RouteForPath(path string) Route {
	for i, char := range path {
		if char == '?' || char == '#' {
			path = path[:i]
			break
		}
	}
	for len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}
	switch path {
	case string(RouteDM), "/":
		return RouteDM
	case string(RoutePhone):
		return RoutePhone
	case string(RouteHost):
		return RouteHost
	case string(RouteAbout):
		return RouteAbout
	default:
		return RouteNotFound
	}
}
