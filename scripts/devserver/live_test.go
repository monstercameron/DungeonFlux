package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceFingerprint_OnlySourceChangesTriggerReload(t *testing.T) {
	root := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("main.go", "package main")
	first, err := sourceFingerprint(root)
	if err != nil {
		t.Fatal(err)
	}
	write("artifacts/build/result.json", "changed")
	write(".git/HEAD", "changed")
	write("TODOS.md", "changed")
	same, err := sourceFingerprint(root)
	if err != nil || same != first {
		t.Fatal("generated output triggered rebuild")
	}
	write("main.go", "package changed")
	changed, err := sourceFingerprint(root)
	if err != nil || changed == first {
		t.Fatal("code edit not detected")
	}
	if err := os.Remove(filepath.Join(root, "main.go")); err != nil {
		t.Fatal(err)
	}
	deleted, err := sourceFingerprint(root)
	if err != nil || deleted == changed {
		t.Fatal("deleted source not detected")
	}
}

func TestInjectLiveReload_HTMLOnlyAndLengthCorrect(t *testing.T) {
	for _, tc := range []struct {
		name, kind, body string
		code             int
		want             bool
	}{
		{"page", "text/html", "<html><body>Demo</body></html>", 200, true},
		{"asset", "application/wasm", "bytes", 200, false},
		{"failure", "text/html", "error", 503, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := &http.Response{StatusCode: tc.code, Header: http.Header{"Content-Type": {tc.kind}, "ETag": {"old"}}, Body: io.NopCloser(strings.NewReader(tc.body))}
			if err := injectLiveReload(response); err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "/__dev/reload.mjs") != tc.want {
				t.Fatalf("body=%s", data)
			}
			if tc.want && (response.ContentLength != int64(len(data)) || response.Header.Get("ETag") != "") {
				t.Fatal("stale response metadata")
			}
		})
	}
}

func TestLiveRoutes_VersionScriptAndRunningChildPreserved(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "web", "splat", "js")
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "dev_reload.mjs"), []byte("// reload"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &supervisor{cfg: configuration{repoRoot: root}, liveVersion: "build-2", child: &exec.Cmd{}}
	child := s.child
	s.startIfIdle("missing.exe")
	if s.child != child {
		t.Fatal("failed rebuild stopped the running child")
	}
	mux := http.NewServeMux()
	s.liveRoutes(mux)
	for _, path := range []string{"/__dev/version", "/__dev/reload.mjs"} {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if path == "/__dev/version" && w.Body.String() != "build-2" {
			t.Fatal("wrong published build")
		}
	}
}

func TestPublishLiveWASM_UpdatesBundleAndRemovesStaleCompression(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bundle")
	goRoot := filepath.Join(root, "go")
	if err := os.MkdirAll(filepath.Join(goRoot, "lib", "wasm"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"dungeonflux.next.wasm": "new", "dungeonflux.wasm": "old", "dungeonflux.wasm.gz": "stale", "dungeonflux.wasm.br": "stale"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"), []byte("runtime"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := publishLiveWASM(dir, goRoot); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "dungeonflux.wasm"))
	if err != nil || string(data) != "new" {
		t.Fatal("new bundle not published")
	}
	for _, ext := range []string{".gz", ".br"} {
		if _, err := os.Stat(filepath.Join(dir, "dungeonflux.wasm"+ext)); !os.IsNotExist(err) {
			t.Fatal("stale compressed bundle retained")
		}
	}
}

func TestLiveReconcile_UnchangedTreeKeepsCurrentProcess(t *testing.T) {
	s := testSupervisor(t)
	s.cfg.liveReload = true
	fingerprint, err := sourceFingerprint(s.cfg.repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	s.liveFingerprint = fingerprint
	s.reconcile()
	if _, err := os.Stat(s.cfg.buildDir); !os.IsNotExist(err) {
		t.Fatal("unchanged source triggered build")
	}
}
