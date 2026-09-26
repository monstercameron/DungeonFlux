package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlaceholderServerRoutesAndRun(t *testing.T) {
	dir := t.TempDir()
	cfg := configuration{port: 18199, phase: "Ready", devlogPath: filepath.Join(dir, "missing.html"), statusPath: filepath.Join(dir, "status.json")}
	w := &statusWriter{path: cfg.statusPath, started: time.Now()}
	server := newPlaceholderServer(cfg, w)
	for _, path := range []string{"/healthz", "/"} {
		recorder := httptest.NewRecorder()
		server.Handler.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
		if recorder.Code != 200 {
			t.Fatalf("%s status = %d", path, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	server.Handler.ServeHTTP(recorder, httptest.NewRequest("GET", "/missing", nil))
	if recorder.Code != 404 {
		t.Fatalf("missing status = %d", recorder.Code)
	}
	if err := run(cfg, func(_ string, _ http.Handler) error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func testSupervisor(t *testing.T) *supervisor {
	t.Helper()
	root := t.TempDir()
	w := &statusWriter{path: filepath.Join(root, "logs", "status.json"), started: time.Now()}
	return &supervisor{cfg: configuration{repoRoot: root, buildDir: filepath.Join(root, "build")}, writer: w}
}

func TestSupervisorHelpers(t *testing.T) {
	root := t.TempDir()
	human := filepath.Join(root, "config", "human.json")
	if err := os.MkdirAll(filepath.Dir(human), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(human, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := defaultConfig(root); got != human {
		t.Fatalf("default config = %q", got)
	}
	if got := absolute(root, "a"); got != filepath.Join(root, "a") {
		t.Fatalf("absolute = %q", got)
	}
	if got := absolute(root, filepath.Join(root, "b")); got != filepath.Join(root, "b") {
		t.Fatalf("absolute path = %q", got)
	}
	if got := defaultConfig(t.TempDir()); !strings.HasSuffix(got, filepath.Join("config", "fake.json")) {
		t.Fatalf("fake config = %q", got)
	}
	if got := tail("short"); got != "short" {
		t.Fatalf("short tail = %q", got)
	}
	if got := tail("1\n2\n3\n4\n5\n6\n7\n8\n9"); got != "2\n3\n4\n5\n6\n7\n8\n9" {
		t.Fatalf("tail = %q", got)
	}
	if got := string(supervisorEnv(root)[0]); !strings.Contains(got, "GOCACHE=") {
		t.Fatalf("env = %q", got)
	}
	if got, err := jsonStatus(status{Mode: "fake"}); err != nil || !strings.Contains(string(got), "fake") {
		t.Fatalf("status = %s, %v", got, err)
	}
}

func TestSupervisorPromoteAndFallback(t *testing.T) {
	s := testSupervisor(t)
	if err := os.MkdirAll(s.cfg.buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	current := filepath.Join(s.cfg.buildDir, "dungeonflux.exe")
	candidate := filepath.Join(s.cfg.buildDir, "dungeonflux.candidate.exe")
	if err := os.WriteFile(current, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(candidate, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.promote(candidate, current); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(current)
	if string(data) != "new" {
		t.Fatalf("current = %q", data)
	}
	old, _ := os.ReadFile(filepath.Join(s.cfg.buildDir, "dungeonflux.last-good.exe"))
	if string(old) != "old" {
		t.Fatalf("last good = %q", old)
	}
	s.startCurrent(filepath.Join(s.cfg.buildDir, "missing.exe"))
	if s.writer.current.Mode != "placeholder" {
		t.Fatalf("missing binary mode = %q", s.writer.current.Mode)
	}
}

func TestSupervisorReconcileBuildsAndPromotes(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	goShim := filepath.Join(binDir, "go.cmd")
	shim := "@copy /Y \"%DF_TEST_EXE%\" \"%~3\" >NUL\r\n"
	if err := os.WriteFile(goShim, []byte(shim), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DF_TEST_EXE", os.Args[0])
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := &supervisor{cfg: configuration{repoRoot: root, buildDir: filepath.Join(root, "build"), dataDir: filepath.Join(root, "data"), configPath: "fake.json", port: 18299, skipGate: true}, writer: &statusWriter{path: filepath.Join(root, "status.json"), started: time.Now()}}
	s.reconcile()
	s.stopChild()
	current := filepath.Join(s.cfg.buildDir, "dungeonflux.exe")
	if _, err := os.Stat(current); err != nil {
		t.Fatalf("current binary: %v", err)
	}
	if s.writer.current.Commit == "" {
		t.Fatal("commit was empty")
	}
}

func TestSupervisorGateSuccess(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "powershell.cmd"), []byte("@exit /b 0\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := testSupervisor(t)
	if err := s.gate(); err != nil {
		t.Fatal(err)
	}
}

func TestSupervisorReconcileFailureKeepsPlaceholder(t *testing.T) {
	s := testSupervisor(t)
	s.cfg.skipGate = true
	s.reconcile()
	if s.writer.current.Mode != "placeholder" || s.child != nil {
		t.Fatalf("mode=%q child=%v", s.writer.current.Mode, s.child)
	}
}

func TestSupervisorStatusAndRouting(t *testing.T) {
	s := testSupervisor(t)
	s.setMode("fake", nil)
	if _, err := os.Stat(s.writer.path); err != nil {
		t.Fatal(err)
	}
	s.fail(errors.New("broken"))
	if s.writer.current.LastErr != "broken" {
		t.Fatalf("last error = %q", s.writer.current.LastErr)
	}
	s.logOutput("test", []byte("output"))
	s.logOutput("empty", nil)
	if data, err := os.ReadFile(filepath.Join(filepath.Dir(s.writer.path), "supervisor.log")); err != nil || !strings.Contains(string(data), "output") {
		t.Fatalf("log = %s, %v", data, err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("GET", "/", nil)
	if s.proxy(recorder, request) {
		t.Fatal("inactive supervisor proxied request")
	}
	s.child = &exec.Cmd{}
	if !s.proxy(httptest.NewRecorder(), request) {
		t.Fatal("active supervisor did not proxy request")
	}
	s.child = nil
	if got := s.commit(); got == "" {
		t.Fatal("empty commit result")
	}
	if done := s.childDone(); done == nil {
		t.Fatal("nil child channel")
	}
	s.restartAfterExit()
	if err := s.build(filepath.Join(t.TempDir(), "candidate.exe")); err == nil {
		t.Fatal("expected build failure in empty repo")
	}
	if err := s.gate(); err == nil {
		t.Fatal("expected gate failure in empty repo")
	}
}

func TestSupervisorStartCurrentRunsOwnedProcess(t *testing.T) {
	s := testSupervisor(t)
	if err := os.MkdirAll(s.cfg.buildDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.cfg.buildDir, "dungeonflux.exe")
	if err := copyFile(os.Args[0], path); err != nil {
		t.Fatal(err)
	}
	s.startCurrent(path)
	s.stopChild()
	if s.child != nil {
		t.Fatal("child was not stopped")
	}
}

func copyFile(source, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o755)
}
