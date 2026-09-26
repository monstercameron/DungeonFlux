package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

// AssetStream receives the chunks for one immutable asset.
type AssetStream interface {
	Recv() (*dungeonfluxv1.AssetChunk, error)
}

// AssetService is the small client surface required by AssetLoader.
type AssetService interface {
	Get(context.Context, *dungeonfluxv1.AssetRequest, ...grpc.CallOption) (AssetStream, error)
	Manifest(context.Context, *dungeonfluxv1.AssetManifestRequest, ...grpc.CallOption) (*dungeonfluxv1.AssetManifestResponse, error)
}

// BlobURLFactory turns transferred bytes into a browser-loadable URL.
type BlobURLFactory interface {
	Create([]byte, string) (string, error)
}

// AssetResult is the asynchronous result of loading one asset.
type AssetResult struct {
	URL string
	Err error
}

// AssetProgress reports one preload completion. Done is true for the final
// notification, including when the manifest or an asset failed.
type AssetProgress struct {
	Completed int
	Total     int
	Name      string
	URL       string
	Err       error
	Done      bool
}

// AssetLoader fetches assets over AssetService and keeps Blob URLs in memory.
// URL lookup is synchronous for render functions; Load and Preload perform all
// network and Blob work on goroutines.
type AssetLoader struct {
	service AssetService
	blobs   BlobURLFactory

	mu       sync.Mutex
	manifest map[string]assetEntry
	bySHA    map[string]string
	urls     map[string]string
	inFlight map[string]*assetFlight
}

type assetEntry struct {
	name        string
	sha256      string
	contentType string
	size        uint64
}

type assetFlight struct {
	done   chan struct{}
	result AssetResult
}

// NewAssetLoader creates a loader with an injected service and Blob factory.
func NewAssetLoader(service AssetService, blobs BlobURLFactory) *AssetLoader {
	return &AssetLoader{
		service:  service,
		blobs:    blobs,
		manifest: make(map[string]assetEntry),
		bySHA:    make(map[string]string),
		urls:     make(map[string]string),
		inFlight: make(map[string]*assetFlight),
	}
}

// ArtURL returns a cached Blob URL for a logical name or SHA-256. It never
// waits; an empty result means the asset is still loading or unavailable.
func (l *AssetLoader) ArtURL(selector string) string {
	if l == nil {
		return ""
	}
	key := l.selectorKey(strings.TrimSpace(selector))
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.urls[key]
}

// Load starts or joins an asynchronous fetch for a logical name or SHA-256.
// Multiple callers for the same manifest asset share one gRPC stream.
func (l *AssetLoader) Load(ctx context.Context, selector string) <-chan AssetResult {
	result := make(chan AssetResult, 1)
	selector = strings.TrimSpace(selector)
	if l == nil {
		result <- AssetResult{Err: errors.New("asset loader is nil")}
		close(result)
		return result
	}
	if ctx == nil {
		result <- AssetResult{Err: errors.New("asset loader: context is nil")}
		close(result)
		return result
	}
	if selector == "" {
		result <- AssetResult{Err: errors.New("asset loader: selector is required")}
		close(result)
		return result
	}
	key := l.selectorKey(selector)
	l.mu.Lock()
	if url := l.urls[key]; url != "" {
		l.mu.Unlock()
		result <- AssetResult{URL: url}
		close(result)
		return result
	}
	flight := l.inFlight[key]
	if flight == nil {
		flight = &assetFlight{done: make(chan struct{})}
		l.inFlight[key] = flight
		go l.fetch(ctx, selector, key, flight)
	}
	l.mu.Unlock()
	go awaitAsset(ctx, flight, result)
	return result
}

// Preload fetches the manifest and every ui/* entry without blocking the
// caller. Progress is called from the preload goroutine after each asset.
func (l *AssetLoader) Preload(ctx context.Context, progress func(AssetProgress)) <-chan error {
	done := make(chan error, 1)
	go func() {
		err := l.preload(ctx, progress)
		done <- err
		close(done)
	}()
	return done
}

func (l *AssetLoader) preload(ctx context.Context, progress func(AssetProgress)) error {
	if l == nil {
		return errors.New("asset loader is nil")
	}
	if ctx == nil {
		return errors.New("asset loader: context is nil")
	}
	if l.service == nil {
		err := errors.New("asset loader: asset service is nil")
		notifyProgress(progress, AssetProgress{Err: err, Done: true})
		return err
	}
	response, err := l.service.Manifest(ctx, &dungeonfluxv1.AssetManifestRequest{})
	if err != nil {
		notifyProgress(progress, AssetProgress{Err: fmt.Errorf("fetch asset manifest: %w", err), Done: true})
		return fmt.Errorf("fetch asset manifest: %w", err)
	}
	entries, err := l.installManifest(response)
	if err != nil {
		notifyProgress(progress, AssetProgress{Err: err, Done: true})
		return err
	}
	uiEntries := make([]assetEntry, 0, len(entries))
	for _, entry := range entries {
		if strings.HasPrefix(entry.name, "ui/") {
			uiEntries = append(uiEntries, entry)
		}
	}
	sort.Slice(uiEntries, func(i, j int) bool { return uiEntries[i].name < uiEntries[j].name })
	loads := make([]preloadLoad, 0, len(uiEntries))
	for _, entry := range uiEntries {
		loads = append(loads, preloadLoad{entry: entry, result: l.Load(ctx, entry.name)})
	}
	var firstErr error
	for completed, load := range loads {
		if ctx.Err() != nil {
			firstErr = ctx.Err()
			break
		}
		loaded := <-load.result
		if loaded.Err != nil && firstErr == nil {
			firstErr = loaded.Err
		}
		notifyProgress(progress, AssetProgress{Completed: completed + 1, Total: len(uiEntries), Name: load.entry.name, URL: loaded.URL, Err: loaded.Err})
	}
	notifyProgress(progress, AssetProgress{Completed: len(uiEntries), Total: len(uiEntries), Err: firstErr, Done: true})
	return firstErr
}

type preloadLoad struct {
	entry  assetEntry
	result <-chan AssetResult
}

func (l *AssetLoader) installManifest(response *dungeonfluxv1.AssetManifestResponse) ([]assetEntry, error) {
	if response == nil {
		return nil, errors.New("asset loader: manifest response is nil")
	}
	entries := make([]assetEntry, 0, len(response.GetAssets()))
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, raw := range response.GetAssets() {
		if raw == nil || strings.TrimSpace(raw.GetName()) == "" || strings.TrimSpace(raw.GetSha256()) == "" {
			return nil, errors.New("asset loader: manifest entry has no name or sha256")
		}
		entry := assetEntry{name: strings.TrimSpace(raw.GetName()), sha256: strings.TrimSpace(raw.GetSha256()), contentType: strings.TrimSpace(raw.GetContentType()), size: raw.GetSize()}
		l.manifest[entry.name] = entry
		l.bySHA[entry.sha256] = entry.name
		entries = append(entries, entry)
	}
	return entries, nil
}

func (l *AssetLoader) fetch(ctx context.Context, selector, key string, flight *assetFlight) {
	result := l.fetchAsset(ctx, selector)
	l.mu.Lock()
	delete(l.inFlight, key)
	flight.result = result
	if result.Err == nil {
		l.cacheResult(selector, key, result.URL)
	}
	l.mu.Unlock()
	close(flight.done)
}

func (l *AssetLoader) fetchAsset(ctx context.Context, selector string) AssetResult {
	if l.service == nil {
		return AssetResult{Err: errors.New("asset loader: asset service is nil")}
	}
	if l.blobs == nil {
		return AssetResult{Err: errors.New("asset loader: Blob factory is nil")}
	}
	request := &dungeonfluxv1.AssetRequest{}
	l.mu.Lock()
	_, knownName := l.manifest[selector]
	_, knownSHA := l.bySHA[selector]
	l.mu.Unlock()
	if (l.isSHA(selector) && !knownName) || knownSHA {
		request.Sha256 = selector
	} else {
		request.Name = selector
	}
	stream, err := l.service.Get(ctx, request)
	if err != nil {
		return AssetResult{Err: fmt.Errorf("fetch asset %q: %w", selector, err)}
	}
	data, contentType, err := readAssetStream(stream)
	if err != nil {
		return AssetResult{Err: fmt.Errorf("read asset %q: %w", selector, err)}
	}
	url, err := l.blobs.Create(data, contentType)
	if err != nil {
		return AssetResult{Err: fmt.Errorf("create Blob URL for %q: %w", selector, err)}
	}
	return AssetResult{URL: url}
}

func readAssetStream(stream AssetStream) ([]byte, string, error) {
	if stream == nil {
		return nil, "", errors.New("asset stream is nil")
	}
	var data []byte
	var contentType string
	var offset uint64
	var expectedSize uint64
	var final bool
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, "", err
		}
		if chunk == nil {
			return nil, "", errors.New("asset stream returned nil chunk")
		}
		if chunk.GetOffset() != offset {
			return nil, "", fmt.Errorf("chunk offset %d, want %d", chunk.GetOffset(), offset)
		}
		if contentType == "" {
			contentType = strings.TrimSpace(chunk.GetContentType())
		}
		if expectedSize == 0 {
			expectedSize = chunk.GetSize()
		}
		data = append(data, chunk.GetData()...)
		offset += uint64(len(chunk.GetData()))
		final = final || chunk.GetFinal()
	}
	if !final {
		return nil, "", errors.New("asset stream ended without final chunk")
	}
	if expectedSize != 0 && expectedSize != uint64(len(data)) {
		return nil, "", fmt.Errorf("asset size %d, received %d", expectedSize, len(data))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return data, contentType, nil
}

func (l *AssetLoader) cacheResult(selector, key, url string) {
	l.urls[key] = url
	selectorKey := l.selectorKeyLocked(selector)
	l.urls[selectorKey] = url
	if entry, ok := l.manifest[selector]; ok {
		l.urls[l.selectorKeyLocked(entry.sha256)] = url
	}
	if name, ok := l.bySHA[selector]; ok {
		l.urls[l.selectorKeyLocked(name)] = url
	}
}

func (l *AssetLoader) selectorKey(selector string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.selectorKeyLocked(selector)
}

func (l *AssetLoader) selectorKeyLocked(selector string) string {
	if entry, ok := l.manifest[selector]; ok {
		return "name:" + entry.name
	}
	if name, ok := l.bySHA[selector]; ok {
		return "name:" + name
	}
	if l.isSHA(selector) {
		return "sha:" + selector
	}
	return "name:" + selector
}

func (l *AssetLoader) isSHA(selector string) bool {
	if len(selector) != 64 {
		return false
	}
	for _, char := range selector {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f') || (char >= 'A' && char <= 'F')) {
			return false
		}
	}
	return true
}

func awaitAsset(ctx context.Context, flight *assetFlight, result chan<- AssetResult) {
	defer close(result)
	select {
	case <-flight.done:
		result <- flight.result
	case <-ctx.Done():
		result <- AssetResult{Err: ctx.Err()}
	}
}

func notifyProgress(progress func(AssetProgress), update AssetProgress) {
	if progress != nil {
		progress(update)
	}
}
