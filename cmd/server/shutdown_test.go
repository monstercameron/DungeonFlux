package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type countingCloser struct {
	calls int
	err   error
}

func (c *countingCloser) Close() error {
	c.calls++
	return c.err
}

func TestShutdown(t *testing.T) {
	tests := []struct {
		name       string
		busy       bool
		appErr     error
		wantForced bool
		wantErr    bool
	}{
		{name: "idle server drains without forcing"},
		{name: "busy download is force-closed and the app still closes", busy: true, wantForced: true},
		{name: "app close error is returned", appErr: errors.New("flush failed"), wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			started := make(chan struct{})
			release := make(chan struct{})
			defer close(release)
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				close(started)
				select { // a response still streaming when the signal arrives
				case <-release:
				case <-r.Context().Done():
				}
			})}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			go func() { _ = server.Serve(listener) }()
			if tc.busy {
				response, err := http.Get("http://" + listener.Addr().String())
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				go func() { _, _ = io.Copy(io.Discard, response.Body) }()
				<-started
			}
			app := &countingCloser{err: tc.appErr}
			var stderr bytes.Buffer
			begin := time.Now()
			err = shutdown(server, app, 50*time.Millisecond, &stderr)
			if (err != nil) != tc.wantErr {
				t.Fatalf("shutdown() error = %v, wantErr %v", err, tc.wantErr)
			}
			if elapsed := time.Since(begin); elapsed > 2*time.Second {
				t.Fatalf("shutdown took %s, want it bounded by the grace period", elapsed)
			}
			if app.calls != 1 {
				t.Fatalf("app closed %d times, want exactly once", app.calls)
			}
			if forced := strings.Contains(stderr.String(), "closing remaining connections"); forced != tc.wantForced {
				t.Fatalf("forced close reported = %v, want %v (stderr %q)", forced, tc.wantForced, stderr.String())
			}
		})
	}
}
