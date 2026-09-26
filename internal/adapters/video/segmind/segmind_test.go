package segmind

import (
	"context"
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
	var downloadCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/video":
			if r.Header.Get("x-api-key") != "secret" {
				t.Errorf("missing API key")
			}
			var payload map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Error(err)
			}
			if !strings.HasPrefix(payload["first_frame_url"].(string), "data:image/png;base64,") || payload["generate_audio"] != false {
				t.Errorf("payload=%v", payload)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(fixture(t, "submit.json"))
		case r.Method == http.MethodGet && r.URL.Path == "/video/seg-123":
			_, _ = w.Write(fixture(t, "done.json"))
		case r.Method == http.MethodGet && r.URL.Path == "/download":
			downloadCalls++
			_, _ = w.Write([]byte("video-bytes"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	adapter := New("secret", server.URL+"/video", httpx.NewVendorClient("segmind", 0, nil))
	job, err := adapter.Submit(t.Context(), ports.VideoRequest{FirstFrame: []byte("png"), LastFrame: []byte("last"), Prompt: "swing", Seconds: 5, Resolution: "480p"})
	if err != nil || job.ID != "seg-123" || job.Vendor != "segmind" {
		t.Fatalf("job=%+v err=%v", job, err)
	}
	status, err := adapter.Poll(t.Context(), job)
	if err != nil || status.State != vocab.JobDone || status.URL == "" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	adapter.endpoint = server.URL
	data, err := adapter.Download(t.Context(), server.URL+"/download")
	if err != nil || string(data) != "video-bytes" || downloadCalls != 1 {
		t.Fatalf("data=%q calls=%d err=%v", data, downloadCalls, err)
	}
}

func TestParseStatus_StatesAndErrors(t *testing.T) {
	tests := []struct {
		name, body string
		state      vocab.JobState
		wantErr    bool
	}{
		{"queued", string(fixture(t, "queued.json")), vocab.JobRunning, false},
		{"running", `{"status":"running"}`, vocab.JobRunning, false},
		{"done", string(fixture(t, "done.json")), vocab.JobDone, false},
		{"failed", `{"status":"failed","error":"blocked"}`, "", true},
		{"unknown", `{"status":"wat"}`, "", true},
		{"done_without_url", `{"status":"completed"}`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := parseStatus([]byte(tt.body))
			if (err != nil) != tt.wantErr || (!tt.wantErr && status.State != tt.state) {
				t.Fatalf("status=%+v err=%v", status, err)
			}
		})
	}
}

func TestBuildSubmitPayload_RejectsMissingInputs(t *testing.T) {
	for _, req := range []ports.VideoRequest{{Seconds: 5, Resolution: "480p"}, {FirstFrame: []byte("x"), Resolution: "480p"}, {FirstFrame: []byte("x"), Seconds: 5}} {
		if _, err := buildSubmitPayload(req); err == nil {
			t.Fatalf("request %+v unexpectedly accepted", req)
		}
	}
}

func TestAdapter_ErrorsAndJobValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusTooManyRequests) }))
	defer server.Close()
	adapter := New("key", server.URL, httpx.NewVendorClient("segmind", 0, nil))
	if _, err := adapter.Submit(context.Background(), ports.VideoRequest{FirstFrame: []byte("x"), Seconds: 4, Resolution: "480p"}); err == nil {
		t.Fatal("expected submit error")
	}
	if _, err := adapter.Poll(context.Background(), ports.VideoJob{}); err == nil {
		t.Fatal("expected empty job error")
	}
	if _, err := adapter.Download(context.Background(), ""); err == nil {
		t.Fatal("expected empty URL error")
	}
}
