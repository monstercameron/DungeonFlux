package main

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"testing"
)

type fakeArtLoader struct {
	mu     sync.Mutex
	cached map[string]string
	loads  []string
	errors []error
}

func (f *fakeArtLoader) ArtURL(selector string) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cached[selector]
}

func (f *fakeArtLoader) Load(_ context.Context, selector string) <-chan AssetResult {
	f.mu.Lock()
	index := len(f.loads)
	f.loads = append(f.loads, selector)
	assetResult := AssetResult{URL: "blob:" + selector}
	if index < len(f.errors) && f.errors[index] != nil {
		assetResult = AssetResult{Err: f.errors[index]}
	} else {
		f.cached[selector] = assetResult.URL
	}
	f.mu.Unlock()
	results := make(chan AssetResult, 1)
	results <- assetResult
	return results
}

func TestNormalizeArtSelector(t *testing.T) {
	tests := []struct{ in, want string }{
		{"/assets/abc123.png", "abc123"},
		{" ui/title_bg ", "ui/title_bg"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := normalizeArtSelector(tc.in); got != tc.want {
			t.Fatalf("normalizeArtSelector(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestLazyArtSource_fetchesOnceAndRefreshes(t *testing.T) {
	loader := &fakeArtLoader{cached: map[string]string{"ui/title_bg": "blob:title"}}
	refreshed := make(chan struct{}, 4)
	source := newLazyArtSource(context.Background(), loader, func() { refreshed <- struct{}{} })

	if got := source.ArtURL("ui/title_bg"); got != "blob:title" {
		t.Fatalf("cached ArtURL = %q", got)
	}
	if got := source.ArtURL("/assets/qrsha.png"); got != "" {
		t.Fatalf("first miss should be empty, got %q", got)
	}
	<-refreshed
	source.ArtURL("/assets/qrsha.png")
	if got := source.ArtURL("/assets/qrsha.png"); got != "blob:qrsha" {
		t.Fatalf("after load ArtURL = %q, want blob:qrsha", got)
	}
	loader.mu.Lock()
	defer loader.mu.Unlock()
	if len(loader.loads) != 1 || loader.loads[0] != "qrsha" {
		t.Fatalf("loads = %v, want exactly one load of qrsha", loader.loads)
	}
}

func TestLazyArtSource_retriesAfterError(t *testing.T) {
	loader := &fakeArtLoader{cached: map[string]string{}, errors: []error{errors.New("temporary miss")}}
	source := newLazyArtSource(context.Background(), loader, nil)
	if got := source.ArtURL("mother_vell"); got != "" {
		t.Fatalf("first miss = %q, want empty", got)
	}
	for attempt := 0; attempt < 1000; attempt++ {
		loader.mu.Lock()
		loads := len(loader.loads)
		loader.mu.Unlock()
		if loads == 2 {
			break
		}
		source.ArtURL("mother_vell")
		runtime.Gosched()
	}
	loader.mu.Lock()
	defer loader.mu.Unlock()
	if len(loader.loads) != 2 {
		t.Fatalf("loads after failed fetch = %v, want retry", loader.loads)
	}
	if got := loader.cached["mother_vell"]; got != "blob:mother_vell" {
		t.Fatalf("retry cache = %q, want successful Blob URL", got)
	}
}
