package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestRestartCombat_DerivesConfigAndStarts(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "fake.json")
	if err := os.WriteFile(configPath, []byte(`{"server":{"debug":true},"debug_start":""}`), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := gotoConfig{configPath: configPath, port: 18101, dataDir: filepath.Join(dir, "data"), pidFile: filepath.Join(dir, "server.pid"), server: "go"}
	var gotName string
	var gotArgs []string
	start := func(name string, args []string, _, _ io.Writer) (*os.Process, error) {
		gotName, gotArgs = name, args
		return os.FindProcess(os.Getpid())
	}
	if err := restartCombat(opts, io.Discard, io.Discard, start); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join("artifacts", "tmp", "L-OPS", "debug-combat.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"debug_start": "combat"`) {
		t.Fatalf("derived config=%s", data)
	}
	if gotName != "go" || len(gotArgs) < 2 || gotArgs[0] != "run" {
		t.Fatalf("start=%q %#v", gotName, gotArgs)
	}
	if !strings.Contains(strings.Join(gotArgs, " "), "-port 18101") {
		t.Fatalf("args=%#v", gotArgs)
	}
}

func TestParseGotoOptionsAndValidation(t *testing.T) {
	opts, err := parseGotoOptions([]string{"--config", "x", "--port", "18102", "--data-dir", "d"}, io.Discard)
	if err != nil || opts.configPath != "x" || opts.port != 18102 || opts.dataDir != "d" {
		t.Fatalf("opts=%#v err=%v", opts, err)
	}
	if _, err := parseGotoOptions([]string{"--port", "0"}, io.Discard); err == nil {
		t.Fatal("invalid port accepted")
	}
	if _, err := parseGotoOptions([]string{"extra"}, io.Discard); err == nil {
		t.Fatal("extra argument accepted")
	}
}

func TestRunGotoRejectsUnknownTarget(t *testing.T) {
	var stderr strings.Builder
	if code, handled := runGoto(context.TODO(), []string{"goto", "opening"}, io.Discard, &stderr); !handled || code != exitTransport || !strings.Contains(stderr.String(), "usage") {
		t.Fatalf("%d %v %q", code, handled, stderr.String())
	}
}

func TestStopOwnedServerRejectsBadPID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pid")
	if err := os.WriteFile(path, []byte(strconv.Itoa(0)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := stopOwnedServer(path); err == nil {
		t.Fatal("bad pid accepted")
	}
}
