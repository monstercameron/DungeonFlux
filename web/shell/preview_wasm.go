//go:build js && wasm

package main

import (
	"net/url"
	"strings"
	"syscall/js"

	dmclient "github.com/monstercameron/DungeonFlux/web/dm"
	phoneclient "github.com/monstercameron/DungeonFlux/web/phone"
	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// PreviewRenderer is the browser-side half of PreviewRegistry. DM and phone
// packages implement it alongside their fixture catalog, without importing
// this package.
type PreviewRenderer interface {
	PreviewRegistry
	Preview(name string) router.Component
}

type dmPreviewRegistry struct{}

func (dmPreviewRegistry) PreviewFixtures() []PreviewFixture {
	fixtures := dmclient.Previews()
	result := make([]PreviewFixture, 0, len(fixtures))
	for name := range fixtures {
		result = append(result, PreviewFixture{Name: name, Label: name})
	}
	return result
}

func (dmPreviewRegistry) Preview(name string) router.Component {
	return func(_ router.Attrs) *router.Element {
		return ui.CreateElement(func(_ struct{}) ui.Node {
			return dmclient.RenderPreview(name, "", nil)
		})
	}
}

type phonePreviewRegistry struct{}

func (phonePreviewRegistry) PreviewFixtures() []PreviewFixture {
	fixtures := phoneclient.Previews()
	result := make([]PreviewFixture, 0, len(fixtures))
	for _, fixture := range fixtures {
		result = append(result, PreviewFixture{Name: fixture.Name, Label: fixture.Name})
	}
	return result
}

func (phonePreviewRegistry) Preview(name string) router.Component {
	return func(_ router.Attrs) *router.Element {
		return ui.CreateElement(func(_ struct{}) ui.Node {
			node, ok := phoneclient.RenderPreview(name)
			if !ok {
				return html.P(html.Props{Role: "alert"}, html.Text("Preview unavailable"))
			}
			return node
		})
	}
}

func dmPreviews() PreviewRenderer { return dmPreviewRegistry{} }

func phonePreviews() PreviewRenderer { return phonePreviewRegistry{} }

// PreviewComponent selects a fixture from the browser query, falling back to
// the normal component when preview mode is absent or the fixture is unknown.
func PreviewComponent(normal router.Component, registry PreviewRenderer) router.Component {
	return func(attrs router.Attrs) *router.Element {
		name := previewName()
		if name == "" || registry == nil {
			return normal(attrs)
		}
		component := registry.Preview(name)
		if component == nil {
			return previewUnavailable(name)
		}
		return component(attrs)
	}
}

func previewUnavailable(name string) *router.Element {
	return html.Main(html.Props{Class: "df-preview-error", Role: "main"},
		html.H1(html.Props{}, html.Text("Preview unavailable")),
		html.P(html.Props{Role: "alert"}, html.Text("No fixture named "+name+" is registered.")),
	)
}

// RegisterPreviewRoute adds the fixture index at /preview.
func RegisterPreviewRoute(parseRouter *router.Router, catalog PreviewCatalog) {
	if parseRouter == nil {
		return
	}
	parseRouter.Register("/preview", func(router.Attrs) *router.Element {
		return previewIndex(catalog)
	})
}

func previewName() string {
	location := js.Global().Get("location")
	if !location.Truthy() {
		return ""
	}
	values, err := url.ParseQuery(location.Get("search").String())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(values.Get("preview"))
}

func previewIndex(catalog PreviewCatalog) *router.Element {
	items := make([]ui.Node, 0, len(catalog.Fixtures()))
	for _, fixture := range catalog.Fixtures() {
		path := "/dm"
		if strings.HasPrefix(fixture.Name, "p:") {
			path = "/p"
		}
		items = append(items, html.Li(html.Props{}, html.A(html.Props{Href: path + "?preview=" + url.QueryEscape(strings.TrimPrefix(strings.TrimPrefix(fixture.Name, "dm:"), "p:"))}, html.Text(fixture.Label))))
	}
	return html.Main(html.Props{Class: "df-preview-index", Role: "main"}, html.H1(html.Props{}, html.Text("Preview fixtures")), html.Ul(html.Props{}, items...))
}
