package elevenlabs

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultEndpoint = "https://api.elevenlabs.io/v1"

// Adapter generates SFX and music with ElevenLabs.
type Adapter struct {
	key      string
	endpoint string
	client   *httpx.Client
}

// New constructs an ElevenLabs sound adapter. Endpoint is injectable for
// tests; an empty endpoint uses the production API root.
func New(key, endpoint string, client *httpx.Client, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if client == nil {
		client = httpx.NewClient(string(vocab.VendorElevenLabs), 30*time.Second, logger)
	}
	return &Adapter{key: key, endpoint: strings.TrimRight(endpoint, "/"), client: client}
}

var _ ports.SoundGen = (*Adapter)(nil)

// Generate performs one sound-generation or music request.
func (a *Adapter) Generate(ctx context.Context, request ports.SoundRequest) (ports.Sound, error) {
	if a == nil || a.client == nil {
		return ports.Sound{}, callError(vocab.ErrUnavailable, errors.New("sound adapter is not configured"))
	}
	path, body, err := buildRequest(request)
	if err != nil {
		return ports.Sound{}, callError(vocab.ErrBadOutput, err)
	}
	httpRequest, err := httpx.ContextRequest(ctx, "POST", a.endpoint+path, strings.NewReader(string(body)))
	if err != nil {
		return ports.Sound{}, callError(vocab.ErrBadOutput, err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if a.key != "" {
		httpRequest.Header.Set("xi-api-key", a.key)
	}
	response, _, err := a.client.Do(httpRequest)
	if err != nil {
		kind := vocab.ErrUnavailable
		if errors.Is(err, context.DeadlineExceeded) {
			kind = vocab.ErrTimeout
		} else if errors.Is(err, context.Canceled) {
			kind = vocab.ErrCanceled
		}
		return ports.Sound{}, callError(kind, err)
	}
	sound, err := parseResponse(response, request.Kind)
	if err != nil {
		return ports.Sound{}, callError(vocab.ErrBadOutput, err)
	}
	sound.DurationMS = int(request.Seconds * 1000)
	return sound, nil
}

func callError(kind vocab.ErrKind, err error) error {
	return &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: kind, Retryable: kind == vocab.ErrUnavailable, Err: err}
}
