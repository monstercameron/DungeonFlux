package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_CompressesInput(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "input.wasm")
	output := filepath.Join(root, "output.wasm.gz")
	want := strings.Repeat("wasm payload\n", 100)
	if err := os.WriteFile(input, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-input", input, "-output", output}); err != nil {
		t.Fatalf("run() error = %v", err)
	}
	file, err := os.Open(output)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(file)
	if err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	data, err := io.ReadAll(reader)
	_ = reader.Close()
	_ = file.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("decompressed output = %q, want %q", data, want)
	}
}

func TestRun_RequiresPaths(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("run() accepted missing paths")
	}
	root := t.TempDir()
	if err := run([]string{"-input", filepath.Join(root, "missing"), "-output", filepath.Join(root, "out")}); err == nil {
		t.Fatal("run() accepted missing output")
	}
	input := filepath.Join(root, "input")
	if err := os.WriteFile(input, []byte("input"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-input", input, "-output", filepath.Join(root, "missing", "out")}); err == nil {
		t.Fatal("run() accepted an unavailable output directory")
	}
	if err := run([]string{"-unknown"}); err == nil {
		t.Fatal("run() accepted an unknown flag")
	}
}
