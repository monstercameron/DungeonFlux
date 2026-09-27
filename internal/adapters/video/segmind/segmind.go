package segmind

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultEndpoint = "https://api.segmind.com/v1/seedance-2.0-mini"

// Adapter is a Segmind Seedance 2.0 Mini VideoGen implementation.
type Adapter struct {
	key      string
	endpoint string
	client   *httpx.Client
}

// New returns an adapter using endpoint, or Segmind's production endpoint when
// endpoint is empty. The API key is supplied by the composition root.
func New(key, endpoint string, client *httpx.Client) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if client == nil {
		client = httpx.NewVendorClient(string(vocab.VendorSegmind), 3*time.Minute, nil)
	}
	return &Adapter{key: key, endpoint: strings.TrimRight(endpoint, "/"), client: client}
}

var _ ports.VideoGen = (*Adapter)(nil)

// Submit starts an asynchronous Seedance generation.
func (a *Adapter) Submit(ctx context.Context, req ports.VideoRequest) (ports.VideoJob, error) {
	body, err := buildSubmitPayload(req)
	if err != nil {
		return ports.VideoJob{}, &ports.CallError{Vendor: vocab.VendorSegmind, Kind: vocab.ErrBadOutput, Err: err}
	}
	httpReq, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint, strings.NewReader(string(body)))
	if err != nil {
		return ports.VideoJob{}, fmt.Errorf("create Segmind request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.key)
	response, _, err := a.client.Do(httpReq)
	if err != nil {
		return ports.VideoJob{}, callError(err, vocab.ErrUnavailable, true)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ports.VideoJob{}, statusError(response.StatusCode, response.Body)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.VideoJob{}, fmt.Errorf("read Segmind submit response: %w", err)
	}
	id, err := parseSubmit(data)
	if err != nil {
		return ports.VideoJob{}, &ports.CallError{Vendor: vocab.VendorSegmind, Kind: vocab.ErrBadOutput, Err: err}
	}
	return ports.VideoJob{Vendor: string(vocab.VendorSegmind), ID: id}, nil
}

// Poll returns the current state of a Segmind job.
func (a *Adapter) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
	if job.ID == "" {
		return ports.VideoStatus{}, &ports.CallError{Vendor: vocab.VendorSegmind, Kind: vocab.ErrBadOutput, Err: fmt.Errorf("job ID is required")}
	}
	httpReq, err := httpx.ContextRequest(ctx, http.MethodGet, a.endpoint+"/"+job.ID, nil)
	if err != nil {
		return ports.VideoStatus{}, fmt.Errorf("create Segmind poll request: %w", err)
	}
	httpReq.Header.Set("x-api-key", a.key)
	response, _, err := a.client.Do(httpReq)
	if err != nil {
		return ports.VideoStatus{}, callError(err, vocab.ErrUnavailable, true)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ports.VideoStatus{}, statusError(response.StatusCode, response.Body)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return ports.VideoStatus{}, fmt.Errorf("read Segmind poll response: %w", err)
	}
	status, err := parseStatus(data)
	if err != nil {
		return ports.VideoStatus{}, &ports.CallError{Vendor: vocab.VendorSegmind, Kind: vocab.ErrBadOutput, Err: err}
	}
	return status, nil
}

// Download retrieves a completed video before the provider's short retention window expires.
func (a *Adapter) Download(ctx context.Context, url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("video URL is required")
	}
	request, err := httpx.ContextRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create Segmind download request: %w", err)
	}
	request.Header.Set("x-api-key", a.key)
	response, _, err := a.client.Do(request)
	if err != nil {
		return nil, callError(err, vocab.ErrUnavailable, true)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, statusError(response.StatusCode, response.Body)
	}
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read Segmind video: %w", err)
	}
	return data, nil
}

func callError(err error, kind vocab.ErrKind, retryable bool) error {
	return &ports.CallError{Vendor: vocab.VendorSegmind, Kind: kind, Retryable: retryable, Err: err}
}

func statusError(status int, body io.Reader) error {
	data, _ := io.ReadAll(io.LimitReader(body, 4096))
	kind := vocab.ErrUnavailable
	retryable := status == http.StatusTooManyRequests || status >= 500
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		kind, retryable = vocab.ErrAuth, false
	}
	if status == http.StatusNotAcceptable {
		kind, retryable = vocab.ErrUnavailable, true
	}
	return &ports.CallError{Vendor: vocab.VendorSegmind, Kind: kind, Retryable: retryable, Err: fmt.Errorf("HTTP %d: %s", status, strings.TrimSpace(string(data)))}
}
