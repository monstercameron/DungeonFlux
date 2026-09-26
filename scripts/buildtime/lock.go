package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	manifestLockRetry = 250 * time.Millisecond
	manifestLockWait  = 60 * time.Second
)

// ManifestLock owns the exclusive lock file protecting manifest.json.
type ManifestLock struct {
	mu   sync.Mutex
	file *os.File
	path string
}

// LockManifest takes an exclusive manifest lock, retrying until it is free.
func LockManifest(ctx context.Context, root string) (*ManifestLock, error) {
	if ctx == nil {
		return nil, errors.New("buildtime: nil lock context")
	}
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("buildtime: wait for manifest lock: %w", ctx.Err())
	default:
	}
	if filepath.Clean(root) == "." || root == "" {
		return nil, errors.New("buildtime: empty manifest directory")
	}
	root = filepath.Clean(root)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("buildtime: create manifest directory: %w", err)
	}
	path := filepath.Join(root, "manifest.lock")
	deadline := time.NewTimer(manifestLockWait)
	defer deadline.Stop()
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			return &ManifestLock{file: file, path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("buildtime: create manifest lock: %w", err)
		}
		retry := time.NewTimer(manifestLockRetry)
		select {
		case <-ctx.Done():
			retry.Stop()
			return nil, fmt.Errorf("buildtime: wait for manifest lock: %w", ctx.Err())
		case <-deadline.C:
			retry.Stop()
			return nil, errors.New("buildtime: manifest lock timeout after 60s")
		case <-retry.C:
		}
	}
}

// UnlockManifest closes and removes a lock owned by the caller.
func UnlockManifest(lock *ManifestLock) error {
	if lock == nil {
		return errors.New("buildtime: nil manifest lock")
	}
	lock.mu.Lock()
	defer lock.mu.Unlock()
	if lock.file == nil {
		return errors.New("buildtime: manifest lock already released")
	}
	closeErr := lock.file.Close()
	lock.file = nil
	removeErr := os.Remove(lock.path)
	if closeErr != nil {
		return fmt.Errorf("buildtime: close manifest lock: %w", closeErr)
	}
	if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("buildtime: remove manifest lock: %w", removeErr)
	}
	return nil
}
