package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSpanishCannedJob_DryRunPlansEveryLine runs the Spanish canned job in
// dry-run mode only: no HTTP client, no paid calls. It proves every English
// line has a non-empty Spanish render under the _es asset convention.
func TestSpanishCannedJob_DryRunPlansEveryLine(t *testing.T) {
	english := CannedLines()
	spanish := SpanishCannedLines()
	if len(spanish) != len(english) {
		t.Fatalf("spanish lines = %d, want %d", len(spanish), len(english))
	}
	byID := make(map[string]CannedLine, len(english))
	for _, line := range english {
		byID[line.ID] = line
	}
	for _, line := range spanish {
		base, ok := byID[strings.TrimSuffix(line.ID, "_es")]
		if !ok || !strings.HasSuffix(line.ID, "_es") {
			t.Errorf("line %q does not map to an english line", line.ID)
		}
		if !strings.HasSuffix(line.Voice, "-es") {
			t.Errorf("line %q voice %q lacks the locale suffix", line.ID, line.Voice)
		}
		if strings.TrimSpace(line.Text) == "" {
			t.Errorf("line %q has empty spanish text", line.ID)
		}
		if ok && line.Text == base.Text {
			t.Errorf("line %q is untranslated", line.ID)
		}
	}
}

func TestSpanishCannedJob_DryRunWritesPlansWithoutNetwork(t *testing.T) {
	output := t.TempDir()
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	job := SpanishCannedJob(nil, "https://example.invalid", output, 1, true)
	if job.Name != "canned-lines-es" {
		t.Fatalf("job name = %q", job.Name)
	}
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	for _, line := range SpanishCannedLines() {
		data, err := os.ReadFile(filepath.Join(output, line.ID+".txt"))
		if err != nil {
			t.Fatalf("plan %s: %v", line.ID, err)
		}
		if strings.TrimSpace(string(data)) != line.Text {
			t.Fatalf("plan %s text drifted", line.ID)
		}
	}
}

func TestSpanishCannedJob_LiveFixtureRegistersEveryLine(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pcm"))
	}))
	defer server.Close()
	root := t.TempDir()
	writer, err := NewManifestWriter(filepath.Join(root, "manifest"))
	if err != nil {
		t.Fatal(err)
	}
	job := SpanishCannedJob(server.Client(), server.URL, filepath.Join(root, "audio"), 1, false)
	if err := job.Run(context.Background(), writer); err != nil {
		t.Fatal(err)
	}
	if len(writer.manifest.Assets) != len(SpanishCannedLines()) {
		t.Fatalf("registered %d Spanish lines, want %d", len(writer.manifest.Assets), len(SpanishCannedLines()))
	}
}

func TestSpanishCannedJob_LiveRequiresClient(t *testing.T) {
	writer, err := NewManifestWriter(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := SpanishCannedJob(nil, "https://example.invalid", t.TempDir(), 1, false).Run(context.Background(), writer); err == nil {
		t.Fatal("Spanish live job accepted nil client")
	}
}
