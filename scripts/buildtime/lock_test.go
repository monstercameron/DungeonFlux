package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWithManifestLock_ReloadsAndPersistsManifest(t *testing.T) {
	root := filepath.Join(t.TempDir(), "buildtime")
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(source, []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.AddFile("old", "AUDIO", source, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write(); err != nil {
		t.Fatal(err)
	}
	if err := withManifestLock(context.Background(), writer, func(current *ManifestWriter) error {
		if _, ok := current.manifest.Assets["old"]; !ok {
			t.Fatal("manifest was not reloaded")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.json")); err != nil {
		t.Fatal(err)
	}
}

func TestWithManifestLock_ReleasesAfterJobError(t *testing.T) {
	root := t.TempDir()
	writer, err := NewManifestWriter(root)
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("job failed")
	if err := withManifestLock(context.Background(), writer, func(*ManifestWriter) error { return want }); !errors.Is(err, want) {
		t.Fatalf("job error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "manifest.lock")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("lock after failed job: %v", err)
	}
}

func TestWithManifestLock_RejectsMissingArguments(t *testing.T) {
	if err := withManifestLock(context.Background(), nil, func(*ManifestWriter) error { return nil }); err == nil {
		t.Fatal("nil writer accepted")
	}
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := withManifestLock(context.Background(), writer, nil); err == nil {
		t.Fatal("nil job accepted")
	}
}

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

func TestLockManifest_RejectsNilContextAndEmptyRoot(t *testing.T) {
	var nilContext context.Context
	//lint:ignore SA1012 this test proves LockManifest rejects a nil context
	if _, err := LockManifest(nilContext, t.TempDir()); err == nil {
		t.Fatal("nil context accepted")
	}
	if _, err := LockManifest(context.Background(), ""); err == nil {
		t.Fatal("empty root accepted")
	}
}
