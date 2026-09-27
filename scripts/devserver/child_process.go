package main

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type stderrTail struct {
	mu      sync.Mutex
	pending string
	last    string
}

func (t *stderrTail) Write(data []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	parts := strings.Split(t.pending+string(data), "\n")
	t.pending = parts[len(parts)-1]
	for _, part := range parts[:len(parts)-1] {
		if line := strings.TrimSpace(part); line != "" {
			t.last = line
		}
	}
	if len(parts) == 1 {
		if line := strings.TrimSpace(parts[0]); line != "" {
			t.last = line
		}
	}
	return len(data), nil
}

func (t *stderrTail) LastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if line := strings.TrimSpace(t.pending); line != "" {
		return line
	}
	return t.last
}

func (s *supervisor) waitChild(cmd interface{ Wait() error }, exit chan<- error, logFile io.Closer, stderr *stderrTail) {
	err := cmd.Wait()
	_ = logFile.Close()
	s.mu.Lock()
	s.lastStderr = stderr.LastLine()
	s.mu.Unlock()
	exit <- err
}

func (s *supervisor) openChildLog() (*os.File, error) {
	dir := filepath.Dir(s.writer.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	stamp := time.Now().UTC().Format("20060102T150405.000000000Z")
	path := filepath.Join(dir, "server-"+stamp+".log")
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
}

func (s *supervisor) childEnvironment() ([]string, error) {
	env := os.Environ()
	for _, item := range env {
		if strings.HasPrefix(item, "DF_DEBUG_TOKEN=") && strings.TrimSpace(strings.TrimPrefix(item, "DF_DEBUG_TOKEN=")) != "" {
			return env, nil
		}
	}
	token, err := newDebugToken()
	if err != nil {
		return nil, err
	}
	if err := writeDebugToken(filepath.Join(s.cfg.dataDir, "debug.token"), token); err != nil {
		return nil, err
	}
	return replaceEnvironment(env, "DF_DEBUG_TOKEN", token), nil
}

func newDebugToken() (string, error) {
	data := make([]byte, 32)
	if _, err := cryptorand.Read(data); err != nil {
		return "", fmt.Errorf("generate debug token: %w", err)
	}
	return hex.EncodeToString(data), nil
}

func writeDebugToken(path, token string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		return fmt.Errorf("write debug token: %w", err)
	}
	return nil
}

func replaceEnvironment(env []string, key, value string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(env)+1)
	for _, item := range env {
		if !strings.HasPrefix(item, prefix) {
			filtered = append(filtered, item)
		}
	}
	return append(filtered, prefix+value)
}
