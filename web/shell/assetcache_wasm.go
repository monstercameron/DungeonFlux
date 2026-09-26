//go:build js && wasm

package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"syscall/js"
	"time"
)

func browserWidth() int {
	window := js.Global().Get("window")
	if !window.Truthy() {
		return 0
	}
	return window.Get("innerWidth").Int()
}

type browserAssetStore struct{}

func newBrowserAssetStore() assetStore { return browserAssetStore{} }

func (browserAssetStore) Get(ctx context.Context, key string) (assetCacheEntry, bool, error) {
	cache, err := browserCache(ctx)
	if err != nil {
		return assetCacheEntry{}, false, err
	}
	response, err := awaitJS(ctx, cache.Call("match", browserRequest(key)))
	if err != nil {
		return assetCacheEntry{}, false, err
	}
	if !response.Truthy() {
		return assetCacheEntry{}, false, nil
	}
	buffer, err := awaitJS(ctx, response.Call("arrayBuffer"))
	if err != nil {
		return assetCacheEntry{}, false, err
	}
	data := make([]byte, buffer.Get("byteLength").Int())
	js.CopyBytesToGo(data, js.Global().Get("Uint8Array").New(buffer))
	headers := response.Get("headers")
	return assetCacheEntry{Data: data, ContentType: browserHeader(headers, "Content-Type"), StoredAt: browserTime(browserHeader(headers, "X-DF-Stored-At")), LastUsed: browserTime(browserHeader(headers, "X-DF-Last-Used"))}, true, nil
}

func (s browserAssetStore) Put(ctx context.Context, key string, entry assetCacheEntry) error {
	cache, err := browserCache(ctx)
	if err != nil {
		return err
	}
	headers := js.Global().Get("Headers").New()
	headers.Call("set", "Content-Type", entry.ContentType)
	headers.Call("set", "X-DF-Stored-At", entry.StoredAt.UTC().Format(time.RFC3339Nano))
	headers.Call("set", "X-DF-Last-Used", entry.LastUsed.UTC().Format(time.RFC3339Nano))
	bytes := js.Global().Get("Uint8Array").New(len(entry.Data))
	js.CopyBytesToJS(bytes, entry.Data)
	options := js.Global().Get("Object").New()
	options.Set("headers", headers)
	response := js.Global().Get("Response").New(bytes, options)
	_, err = awaitJS(ctx, cache.Call("put", browserRequest(key), response))
	return err
}

func (s browserAssetStore) Delete(ctx context.Context, key string) error {
	cache, err := browserCache(ctx)
	if err != nil {
		return err
	}
	_, err = awaitJS(ctx, cache.Call("delete", browserRequest(key)))
	return err
}

func (s browserAssetStore) Keys(ctx context.Context) ([]string, error) {
	cache, err := browserCache(ctx)
	if err != nil {
		return nil, err
	}
	values, err := awaitJS(ctx, cache.Call("keys"))
	if err != nil {
		return nil, err
	}
	keys := make([]string, values.Length())
	for i := range keys {
		raw := values.Index(i).Get("url").String()
		if index := strings.Index(raw, assetCachePrefix); index >= 0 {
			keys[i] = raw[index+len(assetCachePrefix):]
		} else {
			keys[i] = raw
		}
	}
	return keys, nil
}

func (s browserAssetStore) Usage(ctx context.Context) (int64, error) {
	keys, err := s.Keys(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for _, key := range keys {
		if key == manifestCacheKey {
			continue
		}
		entry, ok, err := s.Get(ctx, key)
		if err != nil {
			return 0, err
		}
		if ok {
			total += int64(len(entry.Data))
		}
	}
	return total, nil
}

func browserCache(ctx context.Context) (js.Value, error) {
	caches := js.Global().Get("caches")
	if !caches.Truthy() {
		return js.Undefined(), errors.New("browser Cache Storage is unavailable")
	}
	return awaitJS(ctx, caches.Call("open", assetCacheName))
}

func browserRequest(key string) js.Value {
	return js.Global().Get("Request").New(assetCachePrefix + key)
}

func browserHeader(headers js.Value, key string) string { return headers.Call("get", key).String() }

func browserTime(raw string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func awaitJS(ctx context.Context, promise js.Value) (js.Value, error) {
	if !promise.Truthy() {
		return js.Undefined(), errors.New("browser promise is unavailable")
	}
	result := make(chan jsResult, 1)
	resolve := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			result <- jsResult{value: args[0]}
		}
		return nil
	})
	reject := js.FuncOf(func(_ js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			result <- jsResult{err: fmt.Errorf("browser storage: %s", args[0].String())}
		}
		return nil
	})
	defer resolve.Release()
	defer reject.Release()
	promise.Call("then", resolve).Call("catch", reject)
	select {
	case resolved := <-result:
		if resolved.err != nil {
			return js.Undefined(), resolved.err
		}
		return resolved.value, nil
	case <-ctx.Done():
		return js.Undefined(), ctx.Err()
	}
}

type jsResult struct {
	value js.Value
	err   error
}

func (s browserAssetStore) String() string { return fmt.Sprintf("%T", s) }
