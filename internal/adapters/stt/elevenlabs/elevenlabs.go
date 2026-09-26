package elevenlabs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultEndpoint = "https://api.elevenlabs.io/v1/speech-to-text"

// Adapter implements ports.STT using ElevenLabs Scribe v2 batch transcription.
type Adapter struct {
	key      string
	endpoint string
	client   *httpx.Client
}

// New constructs an ElevenLabs adapter. An empty endpoint uses the public
// Scribe endpoint; endpoint is injectable for tests and compatible proxies.
func New(key, endpoint string, timeout time.Duration, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Adapter{key: key, endpoint: endpoint, client: httpx.NewVendorClient(string(vocab.VendorElevenLabs), timeout, logger)}
}

var _ ports.STT = (*Adapter)(nil)

// Transcribe uploads one released recording and returns its Scribe transcript.
func (a *Adapter) Transcribe(ctx context.Context, req ports.STTRequest) (ports.Transcript, error) {
	body, contentType, err := buildRequest(req)
	if err != nil {
		return ports.Transcript{}, callError(vocab.ErrBadOutput, false, err)
	}
	httpReq, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint, body)
	if err != nil {
		return ports.Transcript{}, callError(vocab.ErrBadOutput, false, err)
	}
	httpReq.Header.Set("xi-api-key", a.key)
	httpReq.Header.Set("Content-Type", contentType)
	response, _, err := a.client.Do(httpReq)
	if err != nil {
		return ports.Transcript{}, classifyTransport(ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return ports.Transcript{}, statusError(response)
	}
	transcript, err := parseResponse(response.Body)
	if err != nil {
		return ports.Transcript{}, callError(vocab.ErrBadOutput, false, err)
	}
	return transcript, nil
}

func statusError(response *http.Response) error {
	message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	kind, retryable := vocab.ErrUnavailable, response.StatusCode >= http.StatusInternalServerError
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		kind = vocab.ErrAuth
	case http.StatusTooManyRequests:
		kind, retryable = vocab.ErrRateLimited, true
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		kind, retryable = vocab.ErrBadOutput, false
	}
	return callError(kind, retryable, fmt.Errorf("status %d: %s", response.StatusCode, strings.TrimSpace(string(message))))
}

func classifyTransport(ctx context.Context, err error) error {
	kind := vocab.ErrUnavailable
	if errors.Is(ctx.Err(), context.Canceled) {
		kind = vocab.ErrCanceled
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		kind = vocab.ErrTimeout
	}
	return callError(kind, kind == vocab.ErrUnavailable, err)
}

func callError(kind vocab.ErrKind, retryable bool, err error) error {
	return &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: kind, Retryable: retryable, Err: err}
}
