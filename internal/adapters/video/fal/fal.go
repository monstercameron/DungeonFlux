package fal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultEndpoint = "https://queue.fal.run"

// Adapter is a fal.ai queue VideoGen implementation.
type Adapter struct {
	key      string
	model    string
	endpoint string
	client   *httpx.Client
	mu       sync.Mutex
	jobs     map[string]queueURLs
}

// New returns an adapter for model. The endpoint is configurable for fixture tests.
func New(key, model, endpoint string, client *httpx.Client) *Adapter {
	if model == "" {
		model = "bytedance/seedance-2.0/fast/image-to-video"
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if client == nil {
		client = httpx.NewVendorClient(string(vocab.VendorFal), 3*time.Minute, nil)
	}
	return &Adapter{key: key, model: model, endpoint: strings.TrimRight(endpoint, "/"), client: client, jobs: make(map[string]queueURLs)}
}

var _ ports.VideoGen = (*Adapter)(nil)

type submitPayload struct {
	Prompt        string `json:"prompt"`
	ImageURL      string `json:"image_url"`
	EndImageURL   string `json:"end_image_url,omitempty"`
	Duration      int    `json:"duration"`
	Resolution    string `json:"resolution"`
	GenerateAudio bool   `json:"generate_audio"`
}

type response struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url"`
	URL       string `json:"url"`
	Error     string `json:"error"`
	QueuePos  int    `json:"queue_position"`
	Response  struct {
		Video struct {
			URL string `json:"url"`
		} `json:"video"`
	} `json:"response"`
}

func payload(req ports.VideoRequest) ([]byte, error) {
	if len(req.FirstFrame) == 0 || req.Seconds <= 0 || req.Resolution == "" {
		return nil, fmt.Errorf("first frame, duration, and resolution are required")
	}
	p := submitPayload{Prompt: req.Prompt, ImageURL: imageData(req.FirstFrame), Duration: req.Seconds, Resolution: req.Resolution, GenerateAudio: false}
	if len(req.LastFrame) > 0 {
		p.EndImageURL = imageData(req.LastFrame)
	}
	return json.Marshal(p)
}

func imageData(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// Submit enqueues a fal.ai video generation.
func (a *Adapter) Submit(ctx context.Context, req ports.VideoRequest) (ports.VideoJob, error) {
	body, err := payload(req)
	if err != nil {
		return ports.VideoJob{}, bad(err)
	}
	request, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint+"/"+a.model, strings.NewReader(string(body)))
	if err != nil {
		return ports.VideoJob{}, fmt.Errorf("create fal request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Key "+a.key)
	data, err := a.doResponse(request)
	if err != nil {
		return ports.VideoJob{}, err
	}
	var result submitResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return ports.VideoJob{}, bad(fmt.Errorf("decode submit response: %w", err))
	}
	if result.RequestID == "" {
		return ports.VideoJob{}, bad(fmt.Errorf("submit response has no request_id"))
	}
	a.mu.Lock()
	a.jobs[result.RequestID] = queueURLs{status: result.StatusURL, response: result.ResponseURL}
	a.mu.Unlock()
	return ports.VideoJob{Vendor: string(vocab.VendorFal), ID: result.RequestID}, nil
}

// Poll returns fal.ai's queue status for a job.
func (a *Adapter) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
	if job.ID == "" {
		return ports.VideoStatus{}, bad(fmt.Errorf("job ID is required"))
	}
	urls := a.urls(job.ID)
	data, err := a.get(ctx, urls.status)
	if err != nil {
		return ports.VideoStatus{}, err
	}
	var status response
	if err := json.Unmarshal(data, &status); err != nil {
		return ports.VideoStatus{}, bad(fmt.Errorf("decode status response: %w", err))
	}
	current, err := state(status.Status)
	if err != nil {
		return ports.VideoStatus{}, bad(err)
	}
	if current == vocab.JobFailed {
		return ports.VideoStatus{}, bad(fmt.Errorf("video job failed: %s", status.Error))
	}
	if current != vocab.JobDone {
		return ports.VideoStatus{State: current, QueuePos: status.QueuePos}, nil
	}
	data, err = a.get(ctx, urls.response)
	if err != nil {
		return ports.VideoStatus{}, err
	}
	url, err := resultURL(data)
	if err != nil {
		return ports.VideoStatus{}, bad(err)
	}
	a.mu.Lock()
	delete(a.jobs, job.ID)
	a.mu.Unlock()
	return ports.VideoStatus{State: vocab.JobDone, URL: url}, nil
}

// Download retrieves the completed video URL returned by fal.ai.
func (a *Adapter) Download(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("video URL is required")
	}
	request, err := httpx.ContextRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create fal download request: %w", err)
	}
	if strings.HasPrefix(url, a.endpoint+"/") {
		request.Header.Set("Authorization", "Key "+a.key)
	}
	return a.doResponse(request)
}

func (a *Adapter) urls(id string) queueURLs {
	a.mu.Lock()
	known, ok := a.jobs[id]
	a.mu.Unlock()
	base := a.endpoint + "/" + appID(a.model) + "/requests/" + id
	if !ok || !strings.HasPrefix(known.status, a.endpoint+"/") {
		known.status = base + "/status"
	}
	if !strings.HasPrefix(known.response, a.endpoint+"/") {
		known.response = base
	}
	return known
}

func (a *Adapter) get(ctx context.Context, url string) ([]byte, error) {
	request, err := httpx.ContextRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create fal poll request: %w", err)
	}
	if strings.HasPrefix(url, a.endpoint+"/") {
		request.Header.Set("Authorization", "Key "+a.key)
	}
	return a.doResponse(request)
}

func (a *Adapter) doResponse(request *http.Request) ([]byte, error) {
	response, _, err := a.client.Do(request)
	if err != nil {
		return nil, &ports.CallError{Vendor: vocab.VendorFal, Kind: vocab.ErrUnavailable, Retryable: true, Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		kind, retryable := vocab.ErrUnavailable, response.StatusCode == 429 || response.StatusCode >= 500
		if response.StatusCode == 401 || response.StatusCode == 403 {
			kind, retryable = vocab.ErrAuth, false
		}
		return nil, &ports.CallError{Vendor: vocab.VendorFal, Kind: kind, Retryable: retryable, Err: fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))}
	}
	return io.ReadAll(response.Body)
}

func parseStatus(data []byte) (ports.VideoStatus, error) {
	var result response
	if err := json.Unmarshal(data, &result); err != nil {
		return ports.VideoStatus{}, bad(fmt.Errorf("decode status response: %w", err))
	}
	state, err := state(result.Status)
	if err != nil {
		return ports.VideoStatus{}, bad(err)
	}
	url := result.VideoURL
	if url == "" {
		url = result.URL
	}
	if url == "" {
		url = result.Response.Video.URL
	}
	if state == vocab.JobDone && url == "" {
		return ports.VideoStatus{}, bad(fmt.Errorf("completed response has no video URL"))
	}
	if state == vocab.JobFailed {
		return ports.VideoStatus{}, bad(fmt.Errorf("video job failed: %s", result.Error))
	}
	return ports.VideoStatus{State: state, QueuePos: result.QueuePos, URL: url}, nil
}

func state(value string) (vocab.JobState, error) {
	switch strings.ToLower(value) {
	case "queued", "pending", "in_queue":
		return vocab.JobQueued, nil
	case "running", "processing", "in_progress":
		return vocab.JobRunning, nil
	case "done", "completed", "success", "succeeded":
		return vocab.JobDone, nil
	case "failed", "error", "canceled", "cancelled":
		return vocab.JobFailed, nil
	default:
		return "", fmt.Errorf("unknown video job status %q", value)
	}
}

func bad(err error) error {
	return &ports.CallError{Vendor: vocab.VendorFal, Kind: vocab.ErrBadOutput, Err: err}
}
