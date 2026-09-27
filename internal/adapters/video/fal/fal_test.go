package fal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAdapter_SubmitPollDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Key secret" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		if r.Method == http.MethodPost {
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if !strings.HasPrefix(body["image_url"].(string), "data:image/png") || body["generate_audio"] != false {
				t.Errorf("body=%v", body)
			}
			if !strings.HasPrefix(body["end_image_url"].(string), "data:image/png;base64,") {
				t.Errorf("missing end_image_url=%v", body)
			}
			if _, present := body["last_frame_url"]; present {
				t.Errorf("obsolete last_frame_url present: %v", body)
			}
			_, _ = w.Write(fixture(t, "submit.json"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/status") {
			_, _ = w.Write(fixture(t, "done.json"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/requests/fal-1") {
			_, _ = w.Write([]byte(`{"video":{"url":"https://cdn.example/fal-1.mp4"}}`))
			return
		}
		_, _ = w.Write([]byte("fal-bytes"))
	}))
	defer server.Close()
	a := New("secret", "test/model", server.URL, httpx.NewVendorClient("fal", 0, nil))
	job, err := a.Submit(t.Context(), ports.VideoRequest{FirstFrame: []byte("png"), LastFrame: []byte("end"), Prompt: "move", Seconds: 5, Resolution: "480p"})
	if err != nil || job.ID != "fal-1" || job.Vendor != "fal" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	status, err := a.Poll(t.Context(), job)
	if err != nil || status.State != vocab.JobDone || status.URL == "" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	data, err := a.Download(t.Context(), server.URL+"/download")
	if err != nil || string(data) != "fal-bytes" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestAdapter_PollResumesRecordedJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Key secret" {
			t.Errorf("authorization=%q", r.Header.Get("Authorization"))
		}
		if strings.HasSuffix(r.URL.Path, "/status") {
			_, _ = w.Write(fixture(t, "done.json"))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/requests/recorded-job") {
			_, _ = w.Write([]byte(`{"video":{"url":"https://cdn.example/recorded.mp4"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()
	a := New("secret", "bytedance/seedance-2.0/fast/image-to-video", server.URL, httpx.NewVendorClient("fal", 0, nil))
	status, err := a.Poll(t.Context(), ports.VideoJob{Vendor: "fal", ID: "recorded-job"})
	if err != nil || status.State != vocab.JobDone || status.URL != "https://cdn.example/recorded.mp4" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
}

func TestAdapter_DownloadDoesNotSendKeyToCDN(t *testing.T) {
	cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("cdn authorization=%q", got)
		}
		_, _ = w.Write([]byte("cdn-video"))
	}))
	defer cdn.Close()
	a := New("secret", "model", "https://queue.example", httpx.NewVendorClient("fal", 0, nil))
	data, err := a.Download(t.Context(), cdn.URL+"/video.mp4")
	if err != nil || string(data) != "cdn-video" {
		t.Fatalf("data=%q err=%v", data, err)
	}
}

func TestParseStatus_Table(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		want       vocab.JobState
		fail       bool
	}{
		{"queued", `{"status":"IN_QUEUE"}`, vocab.JobQueued, false},
		{"running", `{"status":"IN_PROGRESS"}`, vocab.JobRunning, false},
		{"failed", `{"status":"FAILED","error":"bad"}`, "", true},
		{"unknown", `{"status":"other"}`, "", true},
		{"missing_url", `{"status":"COMPLETED"}`, "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseStatus([]byte(tc.body))
			if (err != nil) != tc.fail || (!tc.fail && got.State != tc.want) {
				t.Fatalf("got=%+v err=%v", got, err)
			}
		})
	}
}

func TestPayload_RejectsMissingInputs(t *testing.T) {
	if _, err := payload(ports.VideoRequest{}); err == nil {
		t.Fatal("accepted empty request")
	}
}

func TestAdapter_ErrorPaths(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "limited", http.StatusUnauthorized) }))
	defer server.Close()
	a := New("key", "model", server.URL, httpx.NewVendorClient("fal", 0, nil))
	if _, err := a.Submit(t.Context(), ports.VideoRequest{FirstFrame: []byte("x"), Seconds: 4, Resolution: "480p"}); err == nil {
		t.Fatal("expected HTTP error")
	}
	if _, err := a.Poll(t.Context(), ports.VideoJob{}); err == nil {
		t.Fatal("expected job error")
	}
	if _, err := a.Download(t.Context(), ""); err == nil {
		t.Fatal("expected URL error")
	}
}
