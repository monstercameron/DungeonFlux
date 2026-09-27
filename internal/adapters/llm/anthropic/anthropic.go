package anthropic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	ssestreampkg "github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultModel = "claude-haiku-4-5"

// Adapter implements ports.LLM using the Anthropic Messages streaming API.
type Adapter struct {
	client *anthropic.Client
}

// New constructs an Anthropic adapter. An empty endpoint uses the public API;
// endpoint is injectable for tests and compatible proxies.
func New(key, endpoint string, timeout time.Duration, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = "https://api.anthropic.com"
	}
	httpClient := httpx.NewVendorClient(string(vocab.VendorAnthropic), timeout, logger).HTTPClient()
	client := anthropic.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(strings.TrimRight(endpoint, "/")),
		option.WithHTTPClient(httpClient),
		option.WithMaxRetries(0),
	)
	return &Adapter{client: &client}
}

var _ ports.LLM = (*Adapter)(nil)

// StreamText starts a Claude response stream and returns text deltas as they
// become available. Non-text events are consumed and omitted from the port.
func (a *Adapter) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	params, err := buildParams(req)
	if err != nil {
		return nil, err
	}
	stream := a.client.Messages.NewStreaming(ctx, params)
	if err := stream.Err(); err != nil {
		return nil, classifyError(ctx, err)
	}
	return &streamAdapter{ctx: ctx, stream: stream}, nil
}

// JSON is unsupported because Haiku is reserved for spoken text fallback.
func (a *Adapter) JSON(_ context.Context, _ ports.TextRequest, _ ports.Schema) (json.RawMessage, error) {
	return nil, callError(vocab.ErrUnavailable, false, errors.New("anthropic adapter does not support JSON"))
}

func buildParams(req ports.TextRequest) (anthropic.MessageNewParams, error) {
	if len(req.Messages) == 0 {
		return anthropic.MessageNewParams{}, callError(vocab.ErrBadOutput, false, errors.New("message list is empty"))
	}
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(defaultModel),
		MaxTokens: int64(req.MaxTokens),
	}
	if params.MaxTokens <= 0 {
		params.MaxTokens = 256
	}
	for _, message := range req.Messages {
		switch message.Role {
		case vocab.MsgSystem:
			params.System = append(params.System, anthropic.TextBlockParam{Text: message.Text})
		case vocab.MsgUser:
			params.Messages = append(params.Messages, anthropic.NewUserMessage(anthropic.NewTextBlock(message.Text)))
		case vocab.MsgAssistant:
			params.Messages = append(params.Messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(message.Text)))
		default:
			return anthropic.MessageNewParams{}, callError(vocab.ErrBadOutput, false, fmt.Errorf("unsupported message role %q", message.Role))
		}
	}
	if len(params.Messages) == 0 {
		return anthropic.MessageNewParams{}, callError(vocab.ErrBadOutput, false, errors.New("message list has no user or assistant message"))
	}
	return params, nil
}

type streamAdapter struct {
	ctx    context.Context
	stream *ssestream
}

// Recv returns the next text delta, or io.EOF after the provider stream ends.
func (s *streamAdapter) Recv() (string, error) {
	for s.stream.Next() {
		event := s.stream.Current()
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			return event.Delta.Text, nil
		}
	}
	if err := s.stream.Err(); err != nil {
		return "", classifyError(s.ctx, err)
	}
	return "", io.EOF
}

// Close releases the provider response body.
func (s *streamAdapter) Close() error {
	if err := s.stream.Close(); err != nil {
		return classifyError(s.ctx, err)
	}
	return nil
}

type ssestream = ssestreampkg.Stream[anthropic.MessageStreamEventUnion]

func classifyError(ctx context.Context, err error) error {
	kind, retryable := vocab.ErrUnavailable, true
	if errors.Is(ctx.Err(), context.Canceled) {
		kind, retryable = vocab.ErrCanceled, false
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		kind, retryable = vocab.ErrTimeout, false
	}
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		kind, retryable = statusKind(apiErr.StatusCode)
		return &ports.CallError{Vendor: vocab.VendorAnthropic, Kind: kind, Retryable: retryable, RequestID: apiErr.RequestID, Err: err}
	}
	return callError(kind, retryable, err)
}

func statusKind(status int) (vocab.ErrKind, bool) {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return vocab.ErrAuth, false
	case http.StatusTooManyRequests:
		return vocab.ErrRateLimited, true
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return vocab.ErrBadOutput, false
	case http.StatusRequestTimeout, http.StatusGatewayTimeout:
		return vocab.ErrTimeout, true
	default:
		return vocab.ErrUnavailable, status >= http.StatusInternalServerError
	}
}

func callError(kind vocab.ErrKind, retryable bool, err error) error {
	return &ports.CallError{Vendor: vocab.VendorAnthropic, Kind: kind, Retryable: retryable, Err: err}
}
