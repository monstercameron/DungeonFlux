package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

type supervisor struct {
	cfg        configuration
	writer     *statusWriter
	mu         sync.RWMutex
	child      *exec.Cmd
	childExit  chan error
	lastStderr string
}

func runSupervisor(cfg configuration) error {
	root, err := filepath.Abs(cfg.repoRoot)
	if err != nil {
		return fmt.Errorf("resolve repo: %w", err)
	}
	cfg.repoRoot = root
	cfg.buildDir = absolute(root, cfg.buildDir)
	cfg.dataDir = absolute(root, cfg.dataDir)
	if cfg.configPath == "" {
		cfg.configPath = defaultConfig(root)
	} else {
		cfg.configPath = absolute(root, cfg.configPath)
	}
	started := time.Now()
	writer := &statusWriter{path: absolute(root, cfg.statusPath), started: started, current: status{Mode: "placeholder"}}
	s := &supervisor{cfg: cfg, writer: writer}
	writer.current.Commit = s.commit()
	go s.loop()
	return s.serve()
}

func absolute(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

func defaultConfig(root string) string {
	human := filepath.Join(root, "config", "human.json")
	if _, err := os.Stat(human); err == nil {
		return human
	}
	return filepath.Join(root, "config", "fake.json")
}

func (s *supervisor) loop() {
	interval := s.cfg.interval
	if interval <= 0 {
		interval = 30 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.reconcile()
	for {
		select {
		case <-ticker.C:
			s.reconcile()
		case <-s.childDone():
			s.restartAfterExit()
		}
	}
}

func (s *supervisor) childDone() <-chan error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.childExit != nil {
		return s.childExit
	}
	return make(chan error)
}

func (s *supervisor) reconcile() {
	candidate := filepath.Join(s.cfg.buildDir, "dungeonflux.candidate.exe")
	current := filepath.Join(s.cfg.buildDir, "dungeonflux.exe")
	_ = os.Remove(candidate)
	if err := os.MkdirAll(s.cfg.buildDir, 0o755); err != nil {
		s.fail(err)
		return
	}
	if err := s.build(candidate); err != nil {
		s.fail(err)
		s.startCurrent(current)
		return
	}
	if !s.cfg.skipGate {
		if err := s.gate(); err != nil {
			s.fail(err)
			s.startCurrent(current)
			return
		}
	}
	if err := s.promote(candidate, current); err != nil {
		s.fail(err)
		s.startCurrent(current)
		return
	}
	s.writer.mu.Lock()
	s.writer.current.Commit = s.commit()
	s.writer.current.BuildAt = time.Now().UTC().Format(time.RFC3339)
	s.writer.current.LastErr = ""
	s.writer.mu.Unlock()
	s.startCurrent(current)
}

func (s *supervisor) build(candidate string) error {
	cmd := exec.Command("go", "build", "-o", candidate, "./cmd/server")
	cmd.Dir = s.cfg.repoRoot
	cmd.Env = append(os.Environ(), supervisorEnv(s.cfg.repoRoot)...)
	output, err := cmd.CombinedOutput()
	s.logOutput("build", output)
	if err != nil {
		return fmt.Errorf("build server: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *supervisor) gate() error {
	cmd := exec.Command("powershell", "-NoProfile", "-File", filepath.Join(s.cfg.repoRoot, "scripts", "gate.ps1"), "-Full")
	cmd.Dir = s.cfg.repoRoot
	cmd.Env = append(os.Environ(), supervisorEnv(s.cfg.repoRoot)...)
	output, err := cmd.CombinedOutput()
	s.logOutput("gate", output)
	if err != nil {
		return fmt.Errorf("full gate: %w: %s", err, tail(string(output)))
	}
	return nil
}

func supervisorEnv(root string) []string {
	tmp := filepath.Join(root, "artifacts", "tmp", "ORCH-D")
	cache := filepath.Join(root, "artifacts", "cache", "go")
	return []string{"GOCACHE=" + cache, "GOTMPDIR=" + tmp, "TMP=" + tmp, "TEMP=" + tmp}
}

func (s *supervisor) logOutput(label string, output []byte) {
	if len(output) == 0 {
		return
	}
	path := filepath.Join(filepath.Dir(s.writer.path), "supervisor.log")
	line := fmt.Sprintf("[%s] %s\n", time.Now().UTC().Format(time.RFC3339), label)
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line + string(output))
}
func tail(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) > 8 {
		lines = lines[len(lines)-8:]
	}
	return strings.Join(lines, "\n")
}

func (s *supervisor) promote(candidate, current string) error {
	lastGood := filepath.Join(s.cfg.buildDir, "dungeonflux.last-good.exe")
	s.stopChild()
	if _, err := os.Stat(current); err == nil {
		_ = os.Remove(lastGood)
		if err := os.Rename(current, lastGood); err != nil {
			return fmt.Errorf("save last good: %w", err)
		}
	}
	if err := os.Rename(candidate, current); err != nil {
		return fmt.Errorf("promote candidate: %w", err)
	}
	return nil
}

func (s *supervisor) startCurrent(current string) {
	if _, err := os.Stat(current); err != nil {
		fallback := filepath.Join(s.cfg.buildDir, "dungeonflux.last-good.exe")
		if _, fallbackErr := os.Stat(fallback); fallbackErr != nil {
			s.setMode("placeholder", err)
			return
		}
		current = fallback
	}
	s.stopChild()
	logFile, err := s.openChildLog()
	if err != nil {
		s.fail(fmt.Errorf("open child log: %w", err))
		return
	}
	env, err := s.childEnvironment()
	if err != nil {
		_ = logFile.Close()
		s.fail(fmt.Errorf("prepare child environment: %w", err))
		return
	}
	childPort := s.cfg.port + 1
	args := []string{"-config", s.cfg.configPath, "-port", fmt.Sprint(childPort), "-data-dir", s.cfg.dataDir}
	cmd := exec.Command(current, args...)
	cmd.Dir = s.cfg.repoRoot
	cmd.Env = env
	cmd.Stdout = logFile
	stderr := &stderrTail{}
	cmd.Stderr = io.MultiWriter(logFile, stderr)
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		s.fail(fmt.Errorf("start server: %w", err))
		return
	}
	exit := make(chan error, 1)
	s.mu.Lock()
	s.child = cmd
	s.childExit = exit
	s.lastStderr = ""
	s.mu.Unlock()
	go s.waitChild(cmd, exit, logFile, stderr)
	s.setMode(strings.TrimSuffix(filepath.Base(s.cfg.configPath), filepath.Ext(s.cfg.configPath)), nil)
}

func (s *supervisor) restartAfterExit() {
	s.mu.Lock()
	lastStderr := s.lastStderr
	s.child = nil
	s.childExit = nil
	s.lastStderr = ""
	s.mu.Unlock()
	if lastStderr == "" {
		lastStderr = "server exited; restarting"
	}
	s.setMode("placeholder", errors.New(lastStderr))
	s.startCurrent(filepath.Join(s.cfg.buildDir, "dungeonflux.exe"))
}

func (s *supervisor) stopChild() {
	s.mu.Lock()
	child := s.child
	exit := s.childExit
	s.child = nil
	s.childExit = nil
	s.mu.Unlock()
	if child != nil && child.Process != nil {
		_ = child.Process.Kill()
		if exit != nil {
			<-exit
			return
		}
		_, _ = child.Process.Wait()
	}
}

func (s *supervisor) fail(err error) { s.setMode("placeholder", err) }
func (s *supervisor) setMode(mode string, err error) {
	s.writer.mu.Lock()
	defer s.writer.mu.Unlock()
	s.writer.current.Mode = mode
	if err != nil {
		s.writer.current.LastErr = err.Error()
	}
	_ = s.writer.writeLocked()
}

func (s *supervisor) commit() string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = s.cfg.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func (s *supervisor) serve() error {
	public := fmt.Sprintf(":%d", s.cfg.port)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if s.proxy(w, r) {
			return
		}
		servePlaceholder(w, s.cfg, s.writer)
	})
	server := &http.Server{Addr: public, Handler: mux}
	errs := make(chan error, 1)
	go func() { errs <- server.ListenAndServe() }()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	select {
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		s.stopChild()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	}
}

func (s *supervisor) proxy(w http.ResponseWriter, r *http.Request) bool {
	s.mu.RLock()
	active := s.child != nil
	s.mu.RUnlock()
	if !active {
		return false
	}
	target, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", s.cfg.port+1))
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(rw http.ResponseWriter, _ *http.Request, _ error) {
		http.Error(rw, "server starting", http.StatusServiceUnavailable)
	}
	proxy.ServeHTTP(w, r)
	return true
}

func (s *statusWriter) writeLocked() error {
	s.current.Uptime = time.Since(s.started).Round(time.Second).String()
	data, err := jsonStatus(s.current)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func jsonStatus(value status) ([]byte, error) { return json.MarshalIndent(value, "", "  ") }
