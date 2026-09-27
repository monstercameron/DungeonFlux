package main

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

func TestAssetLoader_PreloadCachesLogicalNameAndSHA(t *testing.T) {
	service := &fakeAssetService{
		manifest: &dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{
			{Name: "ui/title_bg", Sha256: "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e", ContentType: "image/webp", Size: 5},
			{Name: "scene", Sha256: "611511ebb64c2264898a204c561ee1c012789ef0c4244aacd6e1f0243add2fdf", ContentType: "image/webp", Size: 5},
		}},
		assets: map[string]fakeAsset{
			"ui/title_bg": {name: "ui/title_bg", sha: "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e", contentType: "image/webp", data: []byte("title")},
			"scene":       {name: "scene", sha: "611511ebb64c2264898a204c561ee1c012789ef0c4244aacd6e1f0243add2fdf", contentType: "image/webp", data: []byte("scene!")},
		},
	}
	blobs := &fakeBlobURLFactory{}
	loader := NewAssetLoader(service, blobs)
	if got := loader.ArtURL("ui/title_bg"); got != "" {
		t.Fatalf("ArtURL before preload = %q, want empty", got)
	}
	var progress []AssetProgress
	done := loader.Preload(context.Background(), func(update AssetProgress) { progress = append(progress, update) })
	if err := <-done; err != nil {
		t.Fatalf("Preload() error = %v", err)
	}
	if got := loader.ArtURL("ui/title_bg"); got == "" {
		t.Fatal("logical name did not resolve after preload")
	}
	if got := loader.ArtURL("aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e"); got != loader.ArtURL("ui/title_bg") {
		t.Fatalf("SHA URL = %q, logical URL = %q", got, loader.ArtURL("ui/title_bg"))
	}
	if got := loader.ArtURL("scene"); got != "" {
		t.Fatalf("non-ui asset was preloaded as %q", got)
	}
	if blobs.calls != 1 {
		t.Fatalf("Blob calls = %d, want 1", blobs.calls)
	}
	if len(progress) != 2 || !progress[len(progress)-1].Done || progress[0].Completed != 1 || progress[0].Total != 1 {
		t.Fatalf("progress = %#v", progress)
	}
}

func TestAssetLoader_LoadDeduplicatesInFlightFetches(t *testing.T) {
	service := &fakeAssetService{
		assets:     map[string]fakeAsset{"ui/title_bg": {name: "ui/title_bg", sha: "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e", contentType: "image/webp", data: []byte("title")}},
		getStarted: make(chan struct{}),
		releaseGet: make(chan struct{}),
	}
	loader := NewAssetLoader(service, &fakeBlobURLFactory{})
	first := loader.Load(context.Background(), "ui/title_bg")
	<-service.getStarted
	second := loader.Load(context.Background(), "ui/title_bg")
	if got := service.calls(); got != 1 {
		t.Fatalf("Get calls while in flight = %d, want 1", got)
	}
	close(service.releaseGet)
	firstResult := <-first
	secondResult := <-second
	if firstResult.Err != nil || secondResult.Err != nil || firstResult.URL != secondResult.URL {
		t.Fatalf("results = %#v and %#v", firstResult, secondResult)
	}
	if got := service.calls(); got != 1 {
		t.Fatalf("Get calls after completion = %d, want 1", got)
	}
	if got := loader.ArtURL("ui/title_bg"); got != firstResult.URL {
		t.Fatalf("cached URL = %q, want %q", got, firstResult.URL)
	}
}

func TestAssetLoader_LoadReportsValidationAndInputErrors(t *testing.T) {
	tests := []struct {
		name   string
		loader *AssetLoader
		ctx    context.Context
		asset  string
		want   string
	}{
		{name: "nil loader", loader: nil, ctx: context.Background(), asset: "x", want: "loader is nil"},
		{name: "nil context", loader: NewAssetLoader(&fakeAssetService{}, &fakeBlobURLFactory{}), asset: "x", want: "context is nil"},
		{name: "empty selector", loader: NewAssetLoader(&fakeAssetService{}, &fakeBlobURLFactory{}), ctx: context.Background(), want: "selector is required"},
		{name: "nil service", loader: NewAssetLoader(nil, &fakeBlobURLFactory{}), ctx: context.Background(), asset: "x", want: "asset service is nil"},
		{name: "nil Blob factory", loader: NewAssetLoader(&fakeAssetService{}, nil), ctx: context.Background(), asset: "x", want: "Blob factory is nil"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := <-test.loader.Load(test.ctx, test.asset)
			if result.Err == nil || !contains(result.Err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", result.Err, test.want)
			}
		})
	}
}

func TestAssetLoader_PreloadReportsManifestFailure(t *testing.T) {
	wantErr := errors.New("manifest unavailable")
	service := &fakeAssetService{manifestErr: wantErr}
	loader := NewAssetLoader(service, &fakeBlobURLFactory{})
	updates := make(chan AssetProgress, 1)
	err := <-loader.Preload(context.Background(), func(update AssetProgress) { updates <- update })
	if !errors.Is(err, wantErr) {
		t.Fatalf("Preload() error = %v, want %v", err, wantErr)
	}
	update := <-updates
	if !update.Done || !errors.Is(update.Err, wantErr) {
		t.Fatalf("failure progress = %#v", update)
	}
}

func TestAssetLoader_PreloadRejectsInvalidManifest(t *testing.T) {
	loader := NewAssetLoader(&fakeAssetService{manifest: &dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{{Name: "ui/title_bg"}}}}, &fakeBlobURLFactory{})
	if err := <-loader.Preload(context.Background(), nil); err == nil {
		t.Fatal("Preload accepted an entry without SHA-256")
	}
}

func TestReadAssetStream_ValidatesChunks(t *testing.T) {
	tests := []struct {
		name   string
		stream AssetStream
		want   string
	}{
		{name: "nil stream", want: "stream is nil"},
		{name: "nil chunk", stream: &fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{nil}}, want: "nil chunk"},
		{name: "offset gap", stream: &fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{{Offset: 2, Data: []byte("x"), Final: true}}}, want: "offset"},
		{name: "missing final", stream: &fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{{Data: []byte("x")}}}, want: "without final"},
		{name: "size mismatch", stream: &fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{{Size: 2, Data: []byte("x"), Final: true}}}, want: "received"},
		{name: "receive failure", stream: &fakeAssetStream{err: errors.New("stream dropped")}, want: "stream dropped"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := readAssetStream(test.stream)
			if err == nil || !contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
	data, contentType, err := readAssetStream(&fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{{ContentType: "", Data: []byte("x"), Final: true}}})
	if err != nil || string(data) != "x" || contentType != "application/octet-stream" {
		t.Fatalf("default metadata = %q/%q/%v", data, contentType, err)
	}
}

func TestAssetLoader_LoadPreservesVideoContentTypeForBlob(t *testing.T) {
	service := &fakeAssetService{assets: map[string]fakeAsset{
		"opening_clip": {name: "opening_clip", contentType: "video/webm", data: []byte("video")},
	}}
	blobs := &fakeBlobURLFactory{}
	loader := NewAssetLoader(service, blobs)
	result := <-loader.Load(context.Background(), "opening_clip")
	if result.Err != nil || result.URL == "" {
		t.Fatalf("video load = %#v", result)
	}
	if len(blobs.contentTypes) != 1 || blobs.contentTypes[0] != "video/webm" {
		t.Fatalf("Blob content types = %#v, want video/webm", blobs.contentTypes)
	}
}

type fakeAssetService struct {
	mu          sync.Mutex
	manifest    *dungeonfluxv1.AssetManifestResponse
	manifestErr error
	assets      map[string]fakeAsset
	getStarted  chan struct{}
	releaseGet  chan struct{}
	getCount    int
}

type fakeAsset struct {
	name, sha, contentType string
	data                   []byte
}

func (f *fakeAssetService) Manifest(context.Context, *dungeonfluxv1.AssetManifestRequest, ...grpc.CallOption) (*dungeonfluxv1.AssetManifestResponse, error) {
	return f.manifest, f.manifestErr
}

func (f *fakeAssetService) Get(ctx context.Context, request *dungeonfluxv1.AssetRequest, _ ...grpc.CallOption) (AssetStream, error) {
	f.mu.Lock()
	f.getCount++
	if f.getStarted != nil {
		select {
		case <-f.getStarted:
		default:
			close(f.getStarted)
		}
	}
	release := f.releaseGet
	f.mu.Unlock()
	if release != nil {
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	selector := request.GetName()
	if selector == "" {
		selector = request.GetSha256()
	}
	asset, ok := f.assets[selector]
	if !ok {
		return nil, errors.New("asset not found")
	}
	return &fakeAssetStream{chunks: []*dungeonfluxv1.AssetChunk{{Name: asset.name, Sha256: asset.sha, ContentType: asset.contentType, Size: uint64(len(asset.data)), Data: asset.data, Final: true}}}, nil
}

func (f *fakeAssetService) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.getCount
}

type fakeAssetStream struct {
	chunks []*dungeonfluxv1.AssetChunk
	err    error
	index  int
}

func (f *fakeAssetStream) Recv() (*dungeonfluxv1.AssetChunk, error) {
	if f.err != nil {
		err := f.err
		f.err = nil
		return nil, err
	}
	if f.index >= len(f.chunks) {
		return nil, io.EOF
	}
	chunk := f.chunks[f.index]
	f.index++
	return chunk, nil
}

type fakeBlobURLFactory struct {
	mu           sync.Mutex
	calls        int
	contentTypes []string
}

func (f *fakeBlobURLFactory) Create(_ []byte, contentType string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	f.contentTypes = append(f.contentTypes, contentType)
	return "blob:test/" + string(rune('0'+f.calls)), nil
}

func TestAssetLoader_ArtURLFindsNameCachedBeforeManifest(t *testing.T) {
	loader := NewAssetLoader(nil, nil)
	loader.mu.Lock()
	loader.urls["mother_vell"] = "blob:vell"
	loader.mu.Unlock()
	loader.installManifest(&dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{{Name: "mother_vell", Sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ContentType: "image/png"}}})
	if got := loader.ArtURL("mother_vell"); got != "blob:vell" {
		t.Fatalf("ArtURL after manifest = %q, want the blob cached under the name", got)
	}
}
