//go:build js && wasm

package main

import (
	"context"
	"errors"
	"syscall/js"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"github.com/monstercameron/DungeonFlux/web/dm"
	"github.com/monstercameron/DungeonFlux/web/phone"
	"github.com/monstercameron/GoWebComponents/v6/router"
	"google.golang.org/grpc"
)

// NewBrowserAssetLoader creates a Blob URL loader for an AssetService client.
func NewBrowserAssetLoader(client dungeonfluxv1.AssetServiceClient) *AssetLoader {
	if client == nil {
		return NewAssetLoader(nil, browserBlobURLFactory{})
	}
	return NewAssetLoaderWithStore(grpcAssetService{client: client}, browserBlobURLFactory{}, newBrowserAssetStore())
}

type grpcAssetService struct {
	client dungeonfluxv1.AssetServiceClient
}

func (s grpcAssetService) Get(ctx context.Context, request *dungeonfluxv1.AssetRequest, options ...grpc.CallOption) (AssetStream, error) {
	return s.client.Get(ctx, request, options...)
}

func (s grpcAssetService) Manifest(ctx context.Context, request *dungeonfluxv1.AssetManifestRequest, options ...grpc.CallOption) (*dungeonfluxv1.AssetManifestResponse, error) {
	return s.client.Manifest(ctx, request, options...)
}

// installBrowserAssets installs the shared art source and begins manifest
// preloading. The route refresh is scheduled after preload so ArtURL remains
// synchronous during rendering without leaving the initial title art blank.
func installBrowserAssets(ctx context.Context, client *Client) *AssetLoader {
	if client == nil {
		location := js.Global().Get("location")
		if !location.Truthy() {
			return nil
		}
		created, err := NewClient(ctx, location.Get("origin").String()+"/grpc")
		if err != nil {
			logBrowserAssetError(err)
			return nil
		}
		client = created
	}
	if client.conn == nil {
		return nil
	}
	loader := NewBrowserAssetLoader(dungeonfluxv1.NewAssetServiceClient(client.conn))
	lazy := newLazyArtSource(ctx, loader, scheduleAssetRouteRefresh)
	dm.SetArtSource(lazy)
	phone.SetArtSource(lazy)
	loader.Preload(ctx, func(progress AssetProgress) {
		if progress.Err != nil {
			logBrowserAssetError(progress.Err)
		}
		if progress.URL != "" || progress.Done {
			scheduleAssetRouteRefresh()
		}
	})
	return loader
}

func logBrowserAssetError(err error) {
	console := js.Global().Get("console")
	if console.Truthy() && err != nil {
		console.Call("error", "DungeonFlux assets: "+err.Error())
	}
}

type browserBlobURLFactory struct{}

func (browserBlobURLFactory) Create(data []byte, contentType string) (string, error) {
	if !js.Global().Get("Blob").Truthy() || !js.Global().Get("URL").Truthy() {
		return "", errors.New("browser Blob APIs are unavailable")
	}
	bytes := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(bytes, data)
	parts := js.Global().Get("Array").New(0)
	parts.Call("push", bytes)
	options := js.Global().Get("Object").New()
	options.Set("type", contentType)
	blob := js.Global().Get("Blob").New(parts, options)
	url := js.Global().Get("URL").Call("createObjectURL", blob)
	if !url.Truthy() {
		return "", errors.New("browser failed to create Blob URL")
	}
	return url.String(), nil
}

func scheduleAssetRouteRefresh() {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return
	}
	var callback js.Func
	callback = js.FuncOf(func(js.Value, []js.Value) interface{} {
		defer func() { callback.Release() }()
		path := router.GetCurrentPath()
		if path == "" {
			path = string(RouteDM)
		}
		router.Navigate(path)
		return nil
	})
	window.Call("setTimeout", callback, 0)
}
