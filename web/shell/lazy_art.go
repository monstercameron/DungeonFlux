package main

import (
	"context"
	"path"
	"strings"
	"sync"
)

// artLoader is the part of AssetLoader the lazy art source needs.
type artLoader interface {
	ArtURL(selector string) string
	Load(ctx context.Context, selector string) <-chan AssetResult
}

// lazyArtSource resolves art synchronously from the loader's cache and, on a
// miss, starts one background fetch per selector and asks the page to
// re-render when it lands. Runtime assets such as the lobby QR are not part of
// the ui/* preload, so without this they would never be fetched.
type lazyArtSource struct {
	ctx     context.Context
	loader  artLoader
	refresh func()

	mu      sync.Mutex
	started map[string]bool
}

func newLazyArtSource(ctx context.Context, loader artLoader, refresh func()) *lazyArtSource {
	return &lazyArtSource{ctx: ctx, loader: loader, refresh: refresh, started: map[string]bool{}}
}

// ArtURL returns the cached Blob URL for a logical name, a SHA-256, or an
// /assets/<sha256>.<ext> path; it never blocks.
func (s *lazyArtSource) ArtURL(selector string) string {
	if s == nil || s.loader == nil {
		return ""
	}
	selector = normalizeArtSelector(selector)
	if selector == "" {
		return ""
	}
	if url := s.loader.ArtURL(selector); url != "" {
		return url
	}
	s.mu.Lock()
	first := !s.started[selector]
	s.started[selector] = true
	s.mu.Unlock()
	if first {
		go func() {
			result := <-s.loader.Load(s.ctx, selector)
			if result.Err == nil && s.refresh != nil {
				s.refresh()
			}
		}()
	}
	return ""
}

// normalizeArtSelector turns "/assets/<sha256>.<ext>" into the bare SHA-256 and
// leaves logical names untouched.
func normalizeArtSelector(selector string) string {
	selector = strings.TrimSpace(selector)
	if !strings.HasPrefix(selector, "/assets/") {
		return selector
	}
	base := path.Base(selector)
	return strings.TrimSuffix(base, path.Ext(base))
}
