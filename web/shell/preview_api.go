package main

import (
	"sort"
	"strings"
)

// PreviewFixture describes one deterministic browser fixture exposed by a client.
type PreviewFixture struct {
	Name  string
	Label string
}

// PreviewRegistry lists the fixtures owned by one client package.
//
// Client packages implement this small contract locally; they do not import
// the shell package. The WASM-only renderer contract lives in preview_wasm.go.
type PreviewRegistry interface {
	PreviewFixtures() []PreviewFixture
}

// PreviewCatalog combines the fixtures from the DM and phone clients.
type PreviewCatalog struct {
	dm    PreviewRegistry
	phone PreviewRegistry
}

// NewPreviewCatalog creates a catalog from the two client registries.
func NewPreviewCatalog(dm, phone PreviewRegistry) PreviewCatalog {
	return PreviewCatalog{dm: dm, phone: phone}
}

// Fixtures returns all unique fixtures in stable name order, prefixed by client.
func (c PreviewCatalog) Fixtures() []PreviewFixture {
	fixtures := appendCatalogFixtures(nil, "dm", c.dm)
	fixtures = appendCatalogFixtures(fixtures, "p", c.phone)
	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].Name < fixtures[j].Name
	})
	return fixtures
}

func appendCatalogFixtures(dst []PreviewFixture, prefix string, registry PreviewRegistry) []PreviewFixture {
	if registry == nil {
		return dst
	}
	seen := make(map[string]struct{}, len(dst))
	for _, fixture := range dst {
		seen[fixture.Name] = struct{}{}
	}
	for _, fixture := range registry.PreviewFixtures() {
		name := strings.TrimSpace(fixture.Name)
		if name == "" {
			continue
		}
		qualified := prefix + ":" + name
		if _, ok := seen[qualified]; ok {
			continue
		}
		seen[qualified] = struct{}{}
		label := strings.TrimSpace(fixture.Label)
		if label == "" {
			label = name
		}
		dst = append(dst, PreviewFixture{Name: qualified, Label: label})
	}
	return dst
}
