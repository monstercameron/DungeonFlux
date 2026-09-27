package main

import (
	"context"
	"sort"
	"sync"
	"time"
)

const (
	assetCacheName       = "df-assets-v1"
	assetCachePrefix     = "/df-cache/"
	assetTTL             = 30 * 24 * time.Hour
	manifestTTL          = 10 * time.Minute
	phoneAssetCacheCap   = int64(256 << 20)
	desktopAssetCacheCap = int64(1 << 30)
	assetTouchInterval   = time.Hour
)

type assetCacheEntry struct {
	Data        []byte
	ContentType string
	StoredAt    time.Time
	LastUsed    time.Time
}

type assetStore interface {
	Get(context.Context, string) (assetCacheEntry, bool, error)
	Put(context.Context, string, assetCacheEntry) error
	Delete(context.Context, string) error
	Keys(context.Context) ([]string, error)
	Usage(context.Context) (int64, error)
}

type memoryAssetStore struct {
	mu      sync.Mutex
	entries map[string]assetCacheEntry
}

func newMemoryAssetStore() *memoryAssetStore {
	return &memoryAssetStore{entries: make(map[string]assetCacheEntry)}
}

func (s *memoryAssetStore) Get(_ context.Context, key string) (assetCacheEntry, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entries[key]
	if !ok {
		return assetCacheEntry{}, false, nil
	}
	entry.Data = append([]byte(nil), entry.Data...)
	return entry, true, nil
}

func (s *memoryAssetStore) Put(_ context.Context, key string, entry assetCacheEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.Data = append([]byte(nil), entry.Data...)
	s.entries[key] = entry
	return nil
}

func (s *memoryAssetStore) Delete(_ context.Context, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.entries, key)
	return nil
}

func (s *memoryAssetStore) Keys(_ context.Context) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys := make([]string, 0, len(s.entries))
	for key := range s.entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

func (s *memoryAssetStore) Usage(_ context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total int64
	for key, entry := range s.entries {
		if key != manifestCacheKey {
			total += int64(len(entry.Data))
		}
	}
	return total, nil
}

func cacheCap(width int) int64 {
	if width > 0 && width < 700 {
		return phoneAssetCacheCap
	}
	return desktopAssetCacheCap
}

const manifestCacheKey = "manifest"

func pruneAssetStore(ctx context.Context, store assetStore, now time.Time, capBytes int64) error {
	keys, err := store.Keys(ctx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if key == manifestCacheKey {
			continue
		}
		entry, ok, err := store.Get(ctx, key)
		if err != nil {
			return err
		}
		if ok && now.Sub(entry.LastUsed) >= assetTTL {
			if err := store.Delete(ctx, key); err != nil {
				return err
			}
		}
	}
	usage, err := store.Usage(ctx)
	if err != nil || usage <= capBytes {
		return err
	}
	entries := make([]assetLRUEntry, 0, len(keys))
	for _, key := range keys {
		if key == manifestCacheKey {
			continue
		}
		entry, ok, err := store.Get(ctx, key)
		if err != nil {
			return err
		}
		if ok {
			entries = append(entries, assetLRUEntry{key: key, lastUsed: entry.LastUsed, size: int64(len(entry.Data))})
		}
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].lastUsed.Before(entries[j].lastUsed) })
	for _, entry := range entries {
		if usage <= capBytes {
			break
		}
		if err := store.Delete(ctx, entry.key); err != nil {
			return err
		}
		usage -= entry.size
	}
	return nil
}

type assetLRUEntry struct {
	key      string
	lastUsed time.Time
	size     int64
}
