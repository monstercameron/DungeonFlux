package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRunJobs_SortsJobsAndWritesManifest(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	err = RunJobs(context.Background(), writer, []Job{
		{Name: "second", Run: func(context.Context, *ManifestWriter) error { order = append(order, "second"); return nil }},
		{Name: "first", Run: func(context.Context, *ManifestWriter) error { order = append(order, "first"); return nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("jobs ran in %v", order)
	}
}

func TestRunJobs_RejectsInvalidJobsAndCancellation(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		jobs []Job
	}{
		{name: "missing name", jobs: []Job{{Run: func(context.Context, *ManifestWriter) error { return nil }}}},
		{name: "missing function", jobs: []Job{{Name: "job"}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := RunJobs(context.Background(), writer, tc.jobs); err == nil {
				t.Fatal("RunJobs accepted invalid job")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := RunJobs(ctx, writer, []Job{{Name: "job", Run: func(context.Context, *ManifestWriter) error { t.Fatal("job ran"); return nil }}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if err := RunJobs(context.Background(), nil, nil); err == nil {
		t.Fatal("RunJobs accepted nil writer")
	}
}

func TestRunJobs_WrapsJobError(t *testing.T) {
	writer, err := NewManifestWriter(filepath.Join(t.TempDir(), "out"))
	if err != nil {
		t.Fatal(err)
	}
	want := errors.New("failed")
	if err := RunJobs(context.Background(), writer, []Job{{Name: "broken", Run: func(context.Context, *ManifestWriter) error { return want }}}); !errors.Is(err, want) {
		t.Fatalf("expected wrapped job error, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(writer.root), "out", "manifest.json")); !os.IsNotExist(err) {
		t.Fatalf("manifest was written after failed job: %v", err)
	}
}
