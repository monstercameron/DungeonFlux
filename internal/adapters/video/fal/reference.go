package fal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// ReferenceModel is the Seedance 2.0 Fast reference-to-video endpoint used
// for the green-screen combat billboard loops.
const ReferenceModel = "bytedance/seedance-2.0/fast/reference-to-video"

// maxReferenceImages is the endpoint's documented image_urls limit.
const maxReferenceImages = 9

// ReferenceAdapter is a fal.ai queue client for reference-to-video models.
type ReferenceAdapter struct {
	key      string
	model    string
	endpoint string
	client   *httpx.Client
	mu       sync.Mutex
	jobs     map[string]queueURLs
}

type queueURLs struct{ status, response string }

var _ ports.ReferenceVideoGen = (*ReferenceAdapter)(nil)

// NewReference returns a reference-to-video adapter. An empty model selects
// ReferenceModel; an empty endpoint selects the public queue.
func NewReference(key, model, endpoint string, client *httpx.Client) *ReferenceAdapter {
	if model == "" {
		model = ReferenceModel
	}
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if client == nil {
		client = httpx.NewVendorClient(string(vocab.VendorFal), 3*time.Minute, nil)
	}
	return &ReferenceAdapter{key: key, model: model, endpoint: strings.TrimRight(endpoint, "/"), client: client, jobs: make(map[string]queueURLs)}
}

// Model reports the fal model ID the adapter submits to.
func (a *ReferenceAdapter) Model() string { return a.model }

type referencePayload struct {
	Prompt        string   `json:"prompt"`
	ImageURLs     []string `json:"image_urls"`
	Resolution    string   `json:"resolution"`
	Duration      string   `json:"duration"`
	AspectRatio   string   `json:"aspect_ratio,omitempty"`
	GenerateAudio bool     `json:"generate_audio"`
}

type submitResponse struct {
	RequestID   string `json:"request_id"`
	StatusURL   string `json:"status_url"`
	ResponseURL string `json:"response_url"`
}

type resultResponse struct {
	Video struct {
		URL string `json:"url"`
	} `json:"video"`
}

func referenceBody(req ports.ReferenceVideoRequest) ([]byte, error) {
	if strings.TrimSpace(req.Prompt) == "" || req.Seconds <= 0 || req.Resolution == "" {
		return nil, fmt.Errorf("prompt, duration, and resolution are required")
	}
	if len(req.References) == 0 || len(req.References) > maxReferenceImages {
		return nil, fmt.Errorf("reference-to-video needs 1-%d images, got %d", maxReferenceImages, len(req.References))
	}
	urls := make([]string, 0, len(req.References))
	for index, image := range req.References {
		if len(image) == 0 {
			return nil, fmt.Errorf("reference image %d is empty", index+1)
		}
		urls = append(urls, dataURI(image))
	}
	return json.Marshal(referencePayload{Prompt: req.Prompt, ImageURLs: urls, Resolution: req.Resolution, Duration: strconv.Itoa(req.Seconds), AspectRatio: req.Aspect, GenerateAudio: false})
}

// dataURI encodes an image with its sniffed MIME type; fal accepts data URIs
// wherever it accepts image URLs.
func dataURI(data []byte) string {
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		mime = "image/png"
	}
	return "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
}

// SubmitReference enqueues one reference-to-video job.
func (a *ReferenceAdapter) SubmitReference(ctx context.Context, req ports.ReferenceVideoRequest) (ports.VideoJob, error) {
	body, err := referenceBody(req)
	if err != nil {
		return ports.VideoJob{}, bad(err)
	}
	request, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint+"/"+a.model, strings.NewReader(string(body)))
	if err != nil {
		return ports.VideoJob{}, fmt.Errorf("create fal request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Key "+a.key)
	data, err := a.do(request)
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

// Poll reads the queue status; on completion it fetches the result and
// returns the video URL.
func (a *ReferenceAdapter) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
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

func resultURL(data []byte) (string, error) {
	var result resultResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("decode result response: %w", err)
	}
	if result.Video.URL == "" {
		return "", fmt.Errorf("completed response has no video URL")
	}
	return result.Video.URL, nil
}

// urls returns the queue URLs fal returned at submit time, or the documented
// app-level pattern (subpaths are dropped from status and result URLs).
func (a *ReferenceAdapter) urls(id string) queueURLs {
	a.mu.Lock()
	known, ok := a.jobs[id]
	a.mu.Unlock()
	base := a.endpoint + "/" + appID(a.model) + "/requests/" + id
	// The key is sent on these requests, so only URLs on the configured queue
	// host are trusted.
	if !ok || !strings.HasPrefix(known.status, a.endpoint+"/") {
		known.status = base + "/status"
	}
	if !strings.HasPrefix(known.response, a.endpoint+"/") {
		known.response = base
	}
	return known
}

func appID(model string) string {
	parts := strings.Split(strings.Trim(model, "/"), "/")
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, "/")
}

// Download fetches the finished file. fal's CDN URLs are public, so the key
// is only sent to fal's own queue host.
func (a *ReferenceAdapter) Download(ctx context.Context, url string) ([]byte, error) {
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
	return a.do(request)
}

func (a *ReferenceAdapter) get(ctx context.Context, url string) ([]byte, error) {
	request, err := httpx.ContextRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create fal poll request: %w", err)
	}
	request.Header.Set("Authorization", "Key "+a.key)
	return a.do(request)
}

func (a *ReferenceAdapter) do(request *http.Request) ([]byte, error) {
	return (&Adapter{client: a.client}).doResponse(request)
}
