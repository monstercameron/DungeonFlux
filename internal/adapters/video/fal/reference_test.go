package fal

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

var pngHeader = []byte("\x89PNG\r\n\x1a\n0000")

func referenceServer(t *testing.T, statuses []string, polls *atomic.Int32) *httptest.Server {
	t.Helper()
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Errorf("CDN download carried the key")
		}
		_, _ = w.Write([]byte("mp4-bytes"))
	}))
	t.Cleanup(cdn.Close)
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Key secret" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/owner/app/sub/path":
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			images, _ := body["image_urls"].([]any)
			if len(images) != 2 || !strings.HasPrefix(images[0].(string), "data:image/png;base64,") {
				t.Errorf("image_urls=%v", body["image_urls"])
			}
			if body["duration"] != "4" || body["resolution"] != "480p" || body["aspect_ratio"] != "9:16" || body["generate_audio"] != false {
				t.Errorf("body=%v", body)
			}
			_, _ = w.Write([]byte(`{"request_id":"req-1","status_url":"` + server.URL + `/owner/app/requests/req-1/status","response_url":"` + server.URL + `/owner/app/requests/req-1"}`))
		case r.URL.Path == "/owner/app/requests/req-1/status":
			index := int(polls.Add(1)) - 1
			if index >= len(statuses) {
				index = len(statuses) - 1
			}
			_, _ = w.Write([]byte(`{"status":"` + statuses[index] + `","queue_position":3}`))
		case r.URL.Path == "/owner/app/requests/req-1":
			_, _ = w.Write([]byte(`{"video":{"url":"` + cdn.URL + `/cdn/clip.mp4"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	return server
}

func referenceRequest() ports.ReferenceVideoRequest {
	return ports.ReferenceVideoRequest{Prompt: "@Image1 idles", References: [][]byte{pngHeader, pngHeader}, Seconds: 4, Resolution: "480p", Aspect: "9:16"}
}

func TestReferenceAdapter_SubmitPollDownload(t *testing.T) {
	var polls atomic.Int32
	server := referenceServer(t, []string{"IN_QUEUE", "IN_PROGRESS", "COMPLETED"}, &polls)
	defer server.Close()
	a := NewReference("secret", "owner/app/sub/path", server.URL, httpx.NewVendorClient("fal", 0, nil))
	job, err := a.SubmitReference(t.Context(), referenceRequest())
	if err != nil || job.ID != "req-1" || job.Vendor != string(vocab.VendorFal) {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	wantStates := []vocab.JobState{vocab.JobQueued, vocab.JobRunning, vocab.JobDone}
	var status ports.VideoStatus
	for _, want := range wantStates {
		status, err = a.Poll(t.Context(), job)
		if err != nil || status.State != want {
			t.Fatalf("status=%+v err=%v want %s", status, err, want)
		}
	}
	if !strings.HasSuffix(status.URL, "/cdn/clip.mp4") || strings.HasPrefix(status.URL, server.URL) {
		t.Fatalf("url=%q", status.URL)
	}
	data, err := a.Download(t.Context(), status.URL)
	if err != nil || string(data) != "mp4-bytes" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestReferenceAdapter_pollWithoutSubmitUsesAppURL(t *testing.T) {
	var polls atomic.Int32
	server := referenceServer(t, []string{"COMPLETED"}, &polls)
	defer server.Close()
	a := NewReference("secret", "owner/app/sub/path", server.URL, nil)
	status, err := a.Poll(t.Context(), ports.VideoJob{Vendor: "fal", ID: "req-1"})
	if err != nil || status.State != vocab.JobDone {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	if a.Model() != "owner/app/sub/path" || NewReference("k", "", "", nil).Model() != ReferenceModel {
		t.Fatal("model defaults are wrong")
	}
}

func TestReferenceAdapter_rejectsBadRequests(t *testing.T) {
	a := NewReference("secret", "", "http://127.0.0.1:1", nil)
	cases := []struct {
		name string
		req  ports.ReferenceVideoRequest
	}{
		{"no images", ports.ReferenceVideoRequest{Prompt: "p", Seconds: 4, Resolution: "480p"}},
		{"empty image", ports.ReferenceVideoRequest{Prompt: "p", References: [][]byte{nil}, Seconds: 4, Resolution: "480p"}},
		{"too many", ports.ReferenceVideoRequest{Prompt: "p", References: make([][]byte, 10), Seconds: 4, Resolution: "480p"}},
		{"no prompt", ports.ReferenceVideoRequest{References: [][]byte{pngHeader}, Seconds: 4, Resolution: "480p"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := a.SubmitReference(t.Context(), tc.req)
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != vocab.ErrBadOutput {
				t.Fatalf("err=%v", err)
			}
		})
	}
	if _, err := a.Poll(t.Context(), ports.VideoJob{}); err == nil {
		t.Fatal("empty job accepted")
	}
	if _, err := a.Download(t.Context(), ""); err == nil {
		t.Fatal("empty URL accepted")
	}
}

func TestReferenceAdapter_failedAndHTTPErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/fail/status"):
			_, _ = w.Write([]byte(`{"status":"FAILED","error":"nsfw"}`))
		case strings.HasSuffix(r.URL.Path, "/auth/status"):
			w.WriteHeader(http.StatusUnauthorized)
		case strings.HasSuffix(r.URL.Path, "/empty/status"):
			_, _ = w.Write([]byte(`{"status":"COMPLETED"}`))
		case strings.HasSuffix(r.URL.Path, "/empty"):
			_, _ = w.Write([]byte(`{"video":{}}`))
		default:
			w.WriteHeader(http.StatusTooManyRequests)
		}
	}))
	defer server.Close()
	a := NewReference("secret", "owner/app", server.URL, nil)
	cases := []struct {
		id        string
		kind      vocab.ErrKind
		retryable bool
	}{{"fail", vocab.ErrBadOutput, false}, {"auth", vocab.ErrAuth, false}, {"empty", vocab.ErrBadOutput, false}}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			_, err := a.Poll(t.Context(), ports.VideoJob{ID: tc.id})
			var callErr *ports.CallError
			if !errors.As(err, &callErr) || callErr.Kind != tc.kind || callErr.Retryable != tc.retryable {
				t.Fatalf("err=%v", err)
			}
		})
	}
	_, err := a.SubmitReference(t.Context(), referenceRequest())
	var callErr *ports.CallError
	if !errors.As(err, &callErr) || !callErr.Retryable {
		t.Fatalf("429 submit err=%v", err)
	}
}

func TestDataURI_sniffsMIME(t *testing.T) {
	if got := dataURI([]byte("\xff\xd8\xff\xe0jpeg")); !strings.HasPrefix(got, "data:image/jpeg;base64,") {
		t.Fatalf("jpeg=%q", got[:30])
	}
	if got := dataURI([]byte("plain")); !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("fallback=%q", got[:30])
	}
}
