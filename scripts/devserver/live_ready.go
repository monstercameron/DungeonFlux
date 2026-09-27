package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// serveLiveVersion withholds the reload signal until the current child can
// accept requests. Process.Start alone does not establish readiness.
func (s *supervisor) serveLiveVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s.mu.RLock()
	child, version := s.child, s.liveVersion
	s.mu.RUnlock()
	if child == nil || version == "" {
		http.Error(w, "server starting", http.StatusServiceUnavailable)
		return
	}
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/healthz", s.cfg.port+1), nil)
	if err != nil {
		http.Error(w, "server starting", http.StatusServiceUnavailable)
		return
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Do(request)
	if err != nil {
		http.Error(w, "server starting", http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()
	s.mu.RLock()
	ready := response.StatusCode == http.StatusOK && s.child == child && s.liveVersion == version
	s.mu.RUnlock()
	if !ready {
		http.Error(w, "server starting", http.StatusServiceUnavailable)
		return
	}
	_, _ = io.WriteString(w, version)
}

func serveLiveStarting(w http.ResponseWriter, r *http.Request) {
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		http.Error(w, "server starting", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Retry-After", "2")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, `<!doctype html><html lang="en" data-dev-recover><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>DungeonFlux · Updating</title></head><body style="margin:0;min-height:100vh;display:grid;place-items:center;background:#0b1017;color:#efe6d2;font-family:system-ui,sans-serif"><main style="max-width:32rem;padding:2rem;text-align:center"><p style="color:#d9a441;letter-spacing:.16em">DUNGEONFLUX</p><h1>Preparing your table</h1><p role="status">The demo is updating. This page will reconnect automatically.</p></main><script type="module" src="/__dev/reload.mjs"></script></body></html>`)
}
