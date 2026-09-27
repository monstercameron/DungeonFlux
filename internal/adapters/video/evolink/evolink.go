package evolink

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultEndpoint = "https://api.evolink.ai/api/v1/videos/generations"

// Adapter is an EvoLink Seedance 2.0 Mini VideoGen implementation.
type Adapter struct {
	key      string
	endpoint string
	client   *httpx.Client
}

// New returns an adapter using endpoint, or EvoLink's production endpoint when endpoint is empty.
func New(key, endpoint string, client *httpx.Client) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if client == nil {
		client = httpx.NewVendorClient(string(vocab.VendorEvoLink), 3*time.Minute, nil)
	}
	return &Adapter{key: key, endpoint: strings.TrimRight(endpoint, "/"), client: client}
}

var _ ports.VideoGen = (*Adapter)(nil)

type submitPayload struct {
	Model         string `json:"model"`
	Prompt        string `json:"prompt"`
	ImageURL      string `json:"image_url"`
	LastFrameURL  string `json:"last_frame_url,omitempty"`
	Duration      int    `json:"duration"`
	Resolution    string `json:"resolution"`
	GenerateAudio bool   `json:"generate_audio"`
}

type response struct {
	RequestID string `json:"request_id"`
	ID        string `json:"id"`
	Status    string `json:"status"`
	VideoURL  string `json:"video_url"`
	URL       string `json:"url"`
	Output    string `json:"output"`
	QueuePos  int    `json:"queue_position"`
	Error     string `json:"error"`
}

func payload(req ports.VideoRequest) ([]byte, error) {
	if len(req.FirstFrame) == 0 || req.Seconds <= 0 || req.Resolution == "" {
		return nil, fmt.Errorf("first frame, duration, and resolution are required")
	}
	p := submitPayload{Model: "seedance-2.0-mini", Prompt: req.Prompt, ImageURL: imageData(req.FirstFrame), Duration: req.Seconds, Resolution: req.Resolution, GenerateAudio: false}
	if len(req.LastFrame) > 0 {
		p.LastFrameURL = imageData(req.LastFrame)
	}
	return json.Marshal(p)
}

func imageData(data []byte) string {
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
}

// Submit starts an asynchronous EvoLink generation.
func (a *Adapter) Submit(ctx context.Context, req ports.VideoRequest) (ports.VideoJob, error) {
	body, err := payload(req)
	if err != nil {
		return ports.VideoJob{}, bad(err)
	}
	httpReq, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return ports.VideoJob{}, fmt.Errorf("create EvoLink request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+a.key)
	data, err := a.doJSON(httpReq)
	if err != nil {
		return ports.VideoJob{}, err
	}
	var result response
	if err := json.Unmarshal(data, &result); err != nil {
		return ports.VideoJob{}, bad(fmt.Errorf("decode submit response: %w", err))
	}
	if result.RequestID == "" {
		result.RequestID = result.ID
	}
	if result.RequestID == "" {
		return ports.VideoJob{}, bad(fmt.Errorf("submit response has no request_id"))
	}
	return ports.VideoJob{Vendor: string(vocab.VendorEvoLink), ID: result.RequestID}, nil
}

// Poll returns the current state of an EvoLink job.
func (a *Adapter) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
	if job.ID == "" {
		return ports.VideoStatus{}, bad(fmt.Errorf("job ID is required"))
	}
	request, err := httpx.ContextRequest(ctx, http.MethodGet, a.endpoint+"/"+job.ID, nil)
	if err != nil {
		return ports.VideoStatus{}, fmt.Errorf("create EvoLink poll request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+a.key)
	data, err := a.doJSON(request)
	if err != nil {
		return ports.VideoStatus{}, err
	}
	return parseStatus(data)
}

// Download retrieves a completed video from its temporary URL.
func (a *Adapter) Download(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("video URL is required")
	}
	request, err := httpx.ContextRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create EvoLink download request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+a.key)
	data, err := a.doResponse(request)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (a *Adapter) doJSON(request *http.Request) ([]byte, error) { return a.doResponse(request) }

func (a *Adapter) doResponse(request *http.Request) ([]byte, error) {
	response, _, err := a.client.Do(request)
	if err != nil {
		return nil, &ports.CallError{Vendor: vocab.VendorEvoLink, Kind: vocab.ErrUnavailable, Retryable: true, Err: err}
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		kind, retryable := vocab.ErrUnavailable, response.StatusCode == 429 || response.StatusCode >= 500
		if response.StatusCode == 401 || response.StatusCode == 403 {
			kind, retryable = vocab.ErrAuth, false
		}
		return nil, &ports.CallError{Vendor: vocab.VendorEvoLink, Kind: kind, Retryable: retryable, Err: fmt.Errorf("HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))}
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
		url = result.Output
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
	case "queued", "pending", "submitted":
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
	return &ports.CallError{Vendor: vocab.VendorEvoLink, Kind: vocab.ErrBadOutput, Err: err}
}
