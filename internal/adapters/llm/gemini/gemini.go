package gemini

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"google.golang.org/genai"
)

const (
	defaultEndpoint = "https://generativelanguage.googleapis.com/"
	model           = "gemini-3.8-flash"
)

// Adapter implements ports.LLM with Gemini's native GenerateContent API.
type Adapter struct {
	client  *genai.Client
	initErr error
}

// New constructs a Gemini adapter. An empty endpoint uses Google's public API;
// endpoint is injectable for tests and compatible proxies.
func New(key, endpoint string, timeout time.Duration) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:      key,
		Backend:     genai.BackendGeminiAPI,
		HTTPClient:  &http.Client{Timeout: timeout},
		HTTPOptions: genai.HTTPOptions{BaseURL: endpoint},
	})
	return &Adapter{client: client, initErr: err}
}

var _ ports.LLM = (*Adapter)(nil)

// JSON sends a request and returns the model's strict JSON response.
func (a *Adapter) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	config, err := generationConfig(req, schema)
	if err != nil {
		return nil, callError(vocab.ErrBadOutput, false, err)
	}
	response, err := a.client.Models.GenerateContent(ctx, model, contents(req.Messages), config)
	if err != nil {
		return nil, classify(ctx, err)
	}
	text := strings.TrimSpace(response.Text())
	if text == "" {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("gemini response has no text"))
	}
	if !json.Valid([]byte(text)) {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("gemini response is not JSON"))
	}
	return json.RawMessage(text), nil
}

// StreamText sends a text request and exposes its complete response as one
// pull-based stream. Gemini streaming is not used by the demo roles.
func (a *Adapter) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	if err := a.ready(); err != nil {
		return nil, err
	}
	config := &genai.GenerateContentConfig{
		ThinkingConfig:    lowThinking(),
		MaxOutputTokens:   int32(req.MaxTokens),
		SystemInstruction: systemInstruction(req.Messages),
	}
	response, err := a.client.Models.GenerateContent(ctx, model, contents(req.Messages), config)
	if err != nil {
		return nil, classify(ctx, err)
	}
	text := response.Text()
	if text == "" {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("gemini response has no text"))
	}
	return &textStream{text: text}, nil
}

func (a *Adapter) ready() error {
	if a == nil {
		return callError(vocab.ErrUnavailable, false, errors.New("gemini adapter is nil"))
	}
	if a.initErr != nil {
		return callError(vocab.ErrAuth, false, a.initErr)
	}
	if a.client == nil {
		return callError(vocab.ErrUnavailable, false, errors.New("gemini client is nil"))
	}
	return nil
}

func generationConfig(req ports.TextRequest, schema ports.Schema) (*genai.GenerateContentConfig, error) {
	if len(schema.JSON) == 0 {
		return nil, errors.New("JSON schema is empty")
	}
	var value any
	if err := json.Unmarshal(schema.JSON, &value); err != nil {
		return nil, fmt.Errorf("decode JSON schema: %w", err)
	}
	return &genai.GenerateContentConfig{
		ThinkingConfig:     lowThinking(),
		MaxOutputTokens:    int32(req.MaxTokens),
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: value,
		SystemInstruction:  systemInstruction(req.Messages),
	}, nil
}

func lowThinking() *genai.ThinkingConfig {
	return &genai.ThinkingConfig{ThinkingLevel: genai.ThinkingLevelLow}
}

func contents(messages []ports.Message) []*genai.Content {
	result := make([]*genai.Content, 0, len(messages))
	for _, message := range messages {
		if message.Role == vocab.MsgSystem {
			continue
		}
		role := "user"
		if message.Role == vocab.MsgAssistant {
			role = "model"
		}
		result = append(result, &genai.Content{Role: role, Parts: []*genai.Part{{Text: message.Text}}})
	}
	return result
}

func systemInstruction(messages []ports.Message) *genai.Content {
	parts := make([]*genai.Part, 0)
	for _, message := range messages {
		if message.Role == vocab.MsgSystem {
			parts = append(parts, &genai.Part{Text: message.Text})
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return &genai.Content{Parts: parts}
}

func classify(ctx context.Context, err error) error {
	kind, retryable := vocab.ErrUnavailable, true
	if errors.Is(ctx.Err(), context.Canceled) {
		kind, retryable = vocab.ErrCanceled, false
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		kind, retryable = vocab.ErrTimeout, false
	} else {
		var apiErr genai.APIError
		if errors.As(err, &apiErr) {
			switch apiErr.Code {
			case http.StatusUnauthorized, http.StatusForbidden:
				kind, retryable = vocab.ErrAuth, false
			case http.StatusBadRequest:
				kind, retryable = vocab.ErrBadOutput, false
			case http.StatusTooManyRequests:
				kind, retryable = vocab.ErrRateLimited, true
			}
		}
	}
	return callError(kind, retryable, err)
}

func callError(kind vocab.ErrKind, retryable bool, err error) error {
	return &ports.CallError{Vendor: vocab.VendorGemini, Kind: kind, Retryable: retryable, Err: err}
}

type textStream struct {
	text   string
	closed bool
}

func (s *textStream) Recv() (string, error) {
	if s.closed || s.text == "" {
		return "", io.EOF
	}
	text := s.text
	s.text = ""
	return text, nil
}

func (s *textStream) Close() error {
	s.closed = true
	s.text = ""
	return nil
}
