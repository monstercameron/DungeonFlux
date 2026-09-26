package main

import (
	"context"
	"errors"
	"testing"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func TestAssetLoader_ManifestChangeRepointsLogicalName(t *testing.T) {
	loader := NewAssetLoader(nil, &fakeBlobURLFactory{})
	oldSHA := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	newSHA := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	if _, err := loader.installManifest(&dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{{Name: "ui/title_bg", Sha256: oldSHA}}}); err != nil {
		t.Fatal(err)
	}
	loader.urls["name:ui/title_bg"] = "blob:old"
	loader.urls["ui/title_bg"] = "blob:old"
	loader.urls["sha:"+oldSHA] = "blob:old"
	if _, err := loader.installManifest(&dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{{Name: "ui/title_bg", Sha256: newSHA}}}); err != nil {
		t.Fatal(err)
	}
	if got := loader.ArtURL("ui/title_bg"); got != "" {
		t.Fatalf("logical name retained stale URL %q", got)
	}
}

func TestPruneAssetStore_ExpiresAndEvictsLeastRecentlyUsed(t *testing.T) {
	store := newMemoryAssetStore()
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	put := func(key string, size int, used time.Time) {
		t.Helper()
		if err := store.Put(context.Background(), key, assetCacheEntry{Data: make([]byte, size), StoredAt: used, LastUsed: used}); err != nil {
			t.Fatal(err)
		}
	}
	put("expired", 3, now.Add(-assetTTL))
	put("old", 4, now.Add(-2*time.Hour))
	put("new", 4, now.Add(-time.Hour))
	if err := pruneAssetStore(context.Background(), store, now, 5); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.Get(context.Background(), "expired"); ok {
		t.Fatal("expired entry survived pruning")
	}
	if _, ok, _ := store.Get(context.Background(), "old"); ok {
		t.Fatal("least recently used entry survived cap eviction")
	}
	if _, ok, _ := store.Get(context.Background(), "new"); !ok {
		t.Fatal("newest entry was evicted")
	}
}

func TestAssetLoader_LoadPersistentHitAndExpiry(t *testing.T) {
	const sha = "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e"
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	store := newMemoryAssetStore()
	if err := store.Put(context.Background(), sha, assetCacheEntry{Data: []byte("title"), ContentType: "image/webp", StoredAt: now, LastUsed: now}); err != nil {
		t.Fatal(err)
	}
	service := &fakeAssetService{assets: map[string]fakeAsset{sha: {data: []byte("network")}}}
	loader := NewAssetLoaderWithStore(service, &fakeBlobURLFactory{}, store)
	loader.now = func() time.Time { return now.Add(time.Minute) }
	result := <-loader.Load(context.Background(), sha)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if got := service.calls(); got != 0 {
		t.Fatalf("persistent hit made %d network calls", got)
	}
	if got := loader.ArtURL(sha); got != result.URL {
		t.Fatalf("persistent Blob URL = %q, want %q", got, result.URL)
	}
	loader = NewAssetLoaderWithStore(service, &fakeBlobURLFactory{}, store)
	loader.now = func() time.Time { return now.Add(assetTTL) }
	if result := <-loader.Load(context.Background(), sha); result.Err == nil {
		t.Fatal("expired cache entry was served")
	}
}

func TestAssetLoader_PersistentSHAHitRepointsWhenManifestArrives(t *testing.T) {
	const sha = "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e"
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	store := newMemoryAssetStore()
	if err := store.Put(context.Background(), sha, assetCacheEntry{Data: []byte("title"), ContentType: "image/webp", StoredAt: now, LastUsed: now}); err != nil {
		t.Fatal(err)
	}
	loader := NewAssetLoaderWithStore(&fakeAssetService{}, &fakeBlobURLFactory{}, store)
	loader.now = func() time.Time { return now.Add(time.Minute) }
	result := <-loader.Load(context.Background(), sha)
	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if _, err := loader.installManifest(&dungeonfluxv1.AssetManifestResponse{Assets: []*dungeonfluxv1.AssetManifestEntry{{Name: "mother_vell", Sha256: sha, ContentType: "image/webp"}}}); err != nil {
		t.Fatal(err)
	}
	if got := loader.ArtURL("mother_vell"); got != result.URL {
		t.Fatalf("manifest name URL = %q, want %q", got, result.URL)
	}
}

func TestAssetLoader_LoadEvictsSHA256MismatchAndRefetches(t *testing.T) {
	const sha = "aaf2320646108059a87ab5017a86aee454f5378ed95003dbb2e12f4ca5266e0e"
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	store := newMemoryAssetStore()
	_ = store.Put(context.Background(), sha, assetCacheEntry{Data: []byte("tampered"), StoredAt: now, LastUsed: now})
	service := &fakeAssetService{assets: map[string]fakeAsset{sha: {data: []byte("title")}}}
	loader := NewAssetLoaderWithStore(service, &fakeBlobURLFactory{}, store)
	loader.now = func() time.Time { return now.Add(time.Minute) }
	if result := <-loader.Load(context.Background(), sha); result.Err != nil {
		t.Fatal(result.Err)
	}
	if got := service.calls(); got != 1 {
		t.Fatalf("Get calls = %d, want 1", got)
	}
	entry, ok, _ := store.Get(context.Background(), sha)
	if !ok || string(entry.Data) != "title" {
		t.Fatalf("refetched cache = %#v, present=%v", entry, ok)
	}
}

func TestAssetLoader_StorageFailureFallsBackToNetwork(t *testing.T) {
	service := &fakeAssetService{assets: map[string]fakeAsset{"x": {data: []byte("x")}}}
	loader := NewAssetLoaderWithStore(service, &fakeBlobURLFactory{}, failingAssetStore{})
	if result := <-loader.Load(context.Background(), "x"); result.Err != nil {
		t.Fatal(result.Err)
	}
	if result := <-loader.Load(context.Background(), "x"); result.Err != nil {
		t.Fatal(result.Err)
	}
	if got := service.calls(); got != 1 {
		t.Fatalf("fallback memory store made %d network calls, want 1", got)
	}
}

type failingAssetStore struct{}

func (failingAssetStore) Get(context.Context, string) (assetCacheEntry, bool, error) {
	return assetCacheEntry{}, false, errors.New("storage unavailable")
}
func (failingAssetStore) Put(context.Context, string, assetCacheEntry) error {
	return errors.New("storage unavailable")
}
func (failingAssetStore) Delete(context.Context, string) error {
	return errors.New("storage unavailable")
}
func (failingAssetStore) Keys(context.Context) ([]string, error) {
	return nil, errors.New("storage unavailable")
}
func (failingAssetStore) Usage(context.Context) (int64, error) {
	return 0, errors.New("storage unavailable")
}
