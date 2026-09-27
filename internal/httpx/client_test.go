package httpx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestClient_Do_recordsResponseAndTimings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("x-request-id", "req-1")
		_, _ = io.WriteString(w, "hello")
	}))
	defer server.Close()
	client := NewClient("test", time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
	var records []CallStats
	client.SetRecorder(func(stats CallStats) { records = append(records, stats) })
	req, err := ContextRequest(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, stats, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if string(body) != "hello" || stats.Status != http.StatusOK || stats.RequestID != "req-1" {
		t.Fatalf("response=%q stats=%+v", body, stats)
	}
	if len(records) != 1 || records[0].Bytes != 5 || records[0].TTFTMS <= 0 || records[0].DurMS <= 0 {
		t.Fatalf("records=%+v", records)
	}
}

func TestClient_Do_recordsTransportError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := NewVendorClient("test", time.Second, nil)
		wantErr := errors.New("transport unavailable")
		client.HTTPClient().Transport = failingTransport{err: wantErr}
		var record CallStats
		var calls int
		client.SetRecorder(func(stats CallStats) { record = stats; calls++ })
		request, err := ContextRequest(context.Background(), http.MethodGet, "http://example.test", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _, err = client.Do(request)
		if !errors.Is(err, wantErr) {
			t.Fatalf("Do error = %v, want %v", err, wantErr)
		}
		if calls != 1 || record.Vendor != "test" || record.DurMS != 5 || record.Status != 0 {
			t.Fatalf("record=%+v", record)
		}
	})
}

type failingTransport struct{ err error }

func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	<-time.After(5 * time.Millisecond) // synctest advances virtual time only.
	return nil, f.err
}

func TestContextRequest_andClientDefaults(t *testing.T) {
	request, err := ContextRequest(context.Background(), http.MethodPost, "http://example.test", strings.NewReader("x"))
	if err != nil || request.Method != http.MethodPost {
		t.Fatalf("request=%v err=%v", request, err)
	}
	client := NewClient("test", 0, nil)
	if client.HTTPClient().Timeout != 30*time.Second {
		t.Fatalf("timeout=%v", client.HTTPClient().Timeout)
	}
	if errors.Is(nil, context.Canceled) {
		t.Fatal("unexpected nil error match")
	}
}
