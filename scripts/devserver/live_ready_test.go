package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func setLiveTestBackend(t *testing.T, s *supervisor, handler http.Handler) *httptest.Server {
	t.Helper()
	backend := httptest.NewServer(handler)
	t.Cleanup(backend.Close)
	address, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, port, err := net.SplitHostPort(address.Host)
	if err != nil {
		t.Fatal(err)
	}
	number, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	s.cfg.port = number - 1
	return backend
}

func TestLiveVersion_OnlyAdvertisesReadyCurrentChild(t *testing.T) {
	s := &supervisor{child: &exec.Cmd{}, liveVersion: "new-build"}
	var ready atomic.Bool
	backend := setLiveTestBackend(t, s, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/healthz" {
			t.Errorf("unexpected probe: %s", r.URL.Path)
		}
		if !ready.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	check := func(want int, ctx context.Context) {
		t.Helper()
		recorder := httptest.NewRecorder()
		s.serveLiveVersion(recorder, httptest.NewRequest(http.MethodGet, "/__dev/version", nil).WithContext(ctx))
		if recorder.Code != want || recorder.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("version response = %d %v", recorder.Code, recorder.Header())
		}
		if want == 200 && recorder.Body.String() != "new-build" {
			t.Fatal("wrong build version")
		}
		if want != 200 && strings.Contains(recorder.Body.String(), "new-build") {
			t.Fatal("advertised unready build")
		}
	}
	check(503, t.Context())
	ready.Store(true)
	check(200, t.Context())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	check(503, ctx)
	backend.Close()
	check(503, t.Context())
	s.child = nil
	check(503, t.Context())
}

func TestLiveVersion_DoesNotAdvertiseReplacedChild(t *testing.T) {
	s := &supervisor{child: &exec.Cmd{}, liveVersion: "before"}
	setLiveTestBackend(t, s, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		s.mu.Lock()
		s.child, s.liveVersion = &exec.Cmd{}, "after"
		s.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	recorder := httptest.NewRecorder()
	s.serveLiveVersion(recorder, httptest.NewRequest(http.MethodGet, "/__dev/version", nil))
	if recorder.Code != 503 {
		t.Fatalf("replacement returned %d", recorder.Code)
	}
}

func TestLiveStarting_HTMLRecoversAndRPCStaysUnavailable(t *testing.T) {
	for _, tc := range []struct {
		name, accept string
		html         bool
	}{
		{"page", "text/html,application/xhtml+xml", true},
		{"RPC", "application/grpc", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/p", nil)
			request.Header.Set("Accept", tc.accept)
			response := httptest.NewRecorder()
			serveLiveStarting(response, request)
			if response.Code != 503 {
				t.Fatalf("status=%d", response.Code)
			}
			body := response.Body.String()
			if strings.Contains(body, "data-dev-recover") != tc.html || strings.Contains(body, "/__dev/reload.mjs") != tc.html {
				t.Fatalf("recovery page = %s", body)
			}
			if tc.html && (response.Header().Get("Cache-Control") != "no-store" || response.Header().Get("Retry-After") != "2") {
				t.Fatal("startup page can be cached or lacks retry hint")
			}
		})
	}
}

func TestLiveProxy_BackendFailureUsesRecoverablePage(t *testing.T) {
	s := &supervisor{child: &exec.Cmd{}, cfg: configuration{liveReload: true}}
	backend := setLiveTestBackend(t, s, http.NotFoundHandler())
	backend.Close()
	request := httptest.NewRequest(http.MethodGet, "/dm", nil)
	request.Header.Set("Accept", "text/html")
	response := httptest.NewRecorder()
	if !s.proxy(response, request) || response.Code != 503 || !strings.Contains(response.Body.String(), "data-dev-recover") {
		t.Fatalf("proxy failure = %d %s", response.Code, response.Body.String())
	}
}
