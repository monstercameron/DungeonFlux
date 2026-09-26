package main

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_RequiresInputAndOutput(t *testing.T) {
	if err := run(nil); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("error = %v", err)
	}
}

func TestRun_ReportsMissingInput(t *testing.T) {
	err := run([]string{"-input", "missing.spz", "-output", "out.ply"})
	if err == nil || !strings.Contains(err.Error(), "open input") {
		t.Fatalf("error = %v", err)
	}
}

func TestRun_RejectsFlagError(t *testing.T) {
	if err := run([]string{"-unknown"}); err == nil {
		t.Fatal("expected flag error")
	}
}

func TestRun_WritesFullAndLiteOutputs(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "empty.spz")
	var raw bytes.Buffer
	var header [16]byte
	binary.LittleEndian.PutUint32(header[0:4], 0x5053474e)
	binary.LittleEndian.PutUint32(header[4:8], 2)
	raw.Write(header[:])
	var compressed bytes.Buffer
	zw := gzip.NewWriter(&compressed)
	if _, err := zw.Write(raw.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, compressed.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	full, lite := filepath.Join(dir, "full.ply"), filepath.Join(dir, "lite.ply")
	if err := run([]string{"-input", input, "-output", full, "-lite-output", lite, "-max-points", "1"}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{full, lite} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
}
