package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

func (l *AssetLoader) fetchAsset(ctx context.Context, selector string) AssetResult {
	if l.service == nil {
		return AssetResult{Err: errors.New("asset loader: asset service is nil")}
	}
	if l.blobs == nil {
		return AssetResult{Err: errors.New("asset loader: Blob factory is nil")}
	}
	request := &dungeonfluxv1.AssetRequest{}
	l.mu.Lock()
	entry, knownName := l.manifest[selector]
	_, knownSHA := l.bySHA[selector]
	l.mu.Unlock()
	expectedSHA := entry.sha256
	if knownSHA || (l.isSHA(selector) && !knownName) {
		expectedSHA = selector
	}
	if expectedSHA != "" {
		if result, ok := l.loadPersistent(ctx, expectedSHA, entry.contentType); ok {
			// Persistent hits still need the in-memory aliases used by the
			// synchronous ArtURL path. Without this, the Blob URL exists only
			// in the result channel and the phone renders its fallback forever.
			l.mu.Lock()
			l.cacheResult(selector, l.selectorKeyLocked(selector), result.URL)
			l.mu.Unlock()
			return result
		}
	}
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
	actual := assetSHA256(data)
	if expectedSHA != "" && !strings.EqualFold(expectedSHA, actual) {
		return AssetResult{Err: fmt.Errorf("asset %q SHA-256 mismatch: got %s, want %s", selector, actual, expectedSHA)}
	}
	if contentType == "" {
		contentType = entry.contentType
	}
	if err := l.storeAsset(ctx, actual, data, contentType); err != nil {
		l.fallbackStore()
	}
	url, err := l.blobs.Create(data, contentType)
	if err != nil {
		return AssetResult{Err: fmt.Errorf("create Blob URL for %q: %w", selector, err)}
	}
	return AssetResult{URL: url}
}

func assetSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func (l *AssetLoader) loadPersistent(ctx context.Context, sha, contentType string) (AssetResult, bool) {
	if l.store == nil {
		return AssetResult{}, false
	}
	entry, ok, err := l.store.Get(ctx, sha)
	if err != nil {
		l.fallbackStore()
		return AssetResult{}, false
	}
	if !ok {
		return AssetResult{}, false
	}
	now := l.now()
	if entry.LastUsed.IsZero() || now.Sub(entry.LastUsed) >= assetTTL {
		_ = l.store.Delete(ctx, sha)
		return AssetResult{}, false
	}
	digest := sha256.Sum256(entry.Data)
	if !strings.EqualFold(hex.EncodeToString(digest[:]), sha) {
		_ = l.store.Delete(ctx, sha)
		return AssetResult{}, false
	}
	if now.Sub(entry.LastUsed) >= assetTouchInterval {
		entry.LastUsed = now
		_ = l.store.Put(ctx, sha, entry)
	}
	if contentType == "" {
		contentType = entry.ContentType
	}
	url, err := l.blobs.Create(entry.Data, contentType)
	if err != nil {
		return AssetResult{}, false
	}
	return AssetResult{URL: url}, true
}

func (l *AssetLoader) storeAsset(ctx context.Context, sha string, data []byte, contentType string) error {
	if l.store == nil {
		return nil
	}
	now := l.now()
	return l.store.Put(ctx, sha, assetCacheEntry{Data: data, ContentType: contentType, StoredAt: now, LastUsed: now})
}

func (l *AssetLoader) warmManifest(ctx context.Context) error {
	entry, ok, err := l.store.Get(ctx, manifestCacheKey)
	if err != nil || !ok || entry.StoredAt.IsZero() || l.now().Sub(entry.StoredAt) >= manifestTTL {
		return err
	}
	var cached []cachedAssetEntry
	if err := json.Unmarshal(entry.Data, &cached); err != nil {
		_ = l.store.Delete(ctx, manifestCacheKey)
		return err
	}
	response := &dungeonfluxv1.AssetManifestResponse{}
	for _, item := range cached {
		response.Assets = append(response.Assets, &dungeonfluxv1.AssetManifestEntry{Name: item.Name, Sha256: item.Sha256, ContentType: item.ContentType, Size: item.Size})
	}
	_, err = l.installManifest(response)
	return err
}

func (l *AssetLoader) storeManifest(ctx context.Context, entries []assetEntry) error {
	cached := make([]cachedAssetEntry, 0, len(entries))
	for _, entry := range entries {
		cached = append(cached, cachedAssetEntry{Name: entry.name, Sha256: entry.sha256, ContentType: entry.contentType, Size: entry.size})
	}
	data, err := json.Marshal(cached)
	if err != nil {
		return err
	}
	now := l.now()
	return l.store.Put(ctx, manifestCacheKey, assetCacheEntry{Data: data, ContentType: "application/json", StoredAt: now, LastUsed: now})
}

func (l *AssetLoader) fallbackStore() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, memory := l.store.(*memoryAssetStore); !memory {
		l.store = newMemoryAssetStore()
	}
}

type cachedAssetEntry struct {
	Name        string `json:"name"`
	Sha256      string `json:"sha256"`
	ContentType string `json:"content_type,omitempty"`
	Size        uint64 `json:"size,omitempty"`
}
