package main

import "testing"

func TestRouteForPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want Route
	}{
		{name: "dm", path: "/dm", want: RouteDM},
		{name: "root defaults to dm", path: "/", want: RouteDM},
		{name: "phone trailing slash", path: "/p/", want: RoutePhone},
		{name: "host query", path: "/host?debug=1", want: RouteHost},
		{name: "about", path: "/about", want: RouteAbout},
		{name: "empty", path: "", want: RouteNotFound},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := RouteForPath(test.path); got != test.want {
				t.Fatalf("RouteForPath(%q) = %q, want %q", test.path, got, test.want)
			}
		})
	}
}
