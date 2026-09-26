package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManifestLock_ExcludesSecondOwnerUntilReleased(t *testing.T) {
	root := t.TempDir()
	first, err := LockManifest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); err != nil {
		t.Fatalf("lock file missing: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := LockManifest(ctx, root); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second lock error = %v, want deadline exceeded", err)
	}
	if err := UnlockManifest(first); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock file after release: %v", err)
	}
	second, err := LockManifest(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	if err := UnlockManifest(second); err != nil {
		t.Fatal(err)
	}
}

func TestUnlockManifest_RejectsNilAndDoubleRelease(t *testing.T) {
	if err := UnlockManifest(nil); err == nil {
		t.Fatal("nil lock unexpectedly released")
	}
	lock, err := LockManifest(context.Background(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := UnlockManifest(lock); err != nil {
		t.Fatal(err)
	}
	if err := UnlockManifest(lock); err == nil {
		t.Fatal("double release unexpectedly succeeded")
	}
}

func TestLockManifest_RejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := LockManifest(ctx, filepath.Join(t.TempDir(), "nested")); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled lock error = %v", err)
	}
}
