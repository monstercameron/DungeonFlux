package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func sourceFingerprint(root string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && (strings.HasPrefix(entry.Name(), ".") || entry.Name() == "artifacts" || entry.Name() == "dev" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".mjs", ".js", ".html", ".css", ".json", ".proto", ".mod", ".sum":
		default:
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		_, _ = hash.Write([]byte(rel + "\x00"))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (s *supervisor) startIfIdle(current string) {
	s.mu.RLock()
	active := s.child != nil
	s.mu.RUnlock()
	if !active {
		s.startCurrent(current)
	}
}

func (s *supervisor) buildLiveWASM() error {
	dir := filepath.Join(s.cfg.runtimeDir, "artifacts", "wasm")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	candidate := filepath.Join(dir, "dungeonflux.next.wasm")
	cmd := exec.Command("go", "build", "-o", candidate, "./web/shell")
	cmd.Dir = s.cfg.repoRoot
	cmd.Env = append(os.Environ(), supervisorEnv(s.cfg.repoRoot)...)
	cmd.Env = append(cmd.Env, "GOOS=js", "GOARCH=wasm")
	output, err := cmd.CombinedOutput()
	s.logOutput("wasm", output)
	if err != nil {
		return fmt.Errorf("build WASM: %w: %s", err, tail(string(output)))
	}
	goRoot, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return fmt.Errorf("locate Go WASM runtime: %w", err)
	}
	return publishLiveWASM(dir, strings.TrimSpace(string(goRoot)))
}

func publishLiveWASM(dir, goRoot string) error {
	candidate := filepath.Join(dir, "dungeonflux.next.wasm")
	wasmExec, err := os.ReadFile(filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "wasm_exec.js"), wasmExec, 0644); err != nil {
		return err
	}
	if err := os.Rename(candidate, filepath.Join(dir, "dungeonflux.wasm")); err != nil {
		return err
	}
	// These belong to this isolated bundle and must never shadow its new bytes.
	for _, ext := range []string{".gz", ".br"} {
		if err := os.Remove(filepath.Join(dir, "dungeonflux.wasm"+ext)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func (s *supervisor) liveRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/__dev/version", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		s.mu.RLock()
		version := s.liveVersion
		s.mu.RUnlock()
		_, _ = io.WriteString(w, version)
	})
	mux.HandleFunc("/__dev/reload.mjs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-store")
		http.ServeFile(w, r, filepath.Join(s.cfg.repoRoot, "web", "splat", "js", "dev_reload.mjs"))
	})
}

func injectLiveReload(response *http.Response) error {
	if !strings.Contains(response.Header.Get("Content-Type"), "text/html") || response.StatusCode != http.StatusOK {
		return nil
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if err := response.Body.Close(); err != nil {
		return err
	}
	data = bytes.Replace(data, []byte("</body>"), []byte(`<script type="module" src="/__dev/reload.mjs"></script></body>`), 1)
	response.Body = io.NopCloser(bytes.NewReader(data))
	response.ContentLength = int64(len(data))
	response.Header.Set("Content-Length", strconv.Itoa(len(data)))
	response.Header.Del("ETag")
	response.Header.Set("Cache-Control", "no-store")
	return nil
}
