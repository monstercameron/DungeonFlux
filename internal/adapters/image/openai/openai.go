package openai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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

const (
	defaultEndpoint = "https://api.openai.com/v1/images/generations"
	model           = "gpt-image-2.5-flare"
)

// Adapter implements ports.ImageGen using the OpenAI Images API.
type Adapter struct {
	key      string
	endpoint string
	client   *httpx.Client
}

// New constructs an OpenAI image adapter. An empty endpoint uses the public
// OpenAI Images API; endpoint is injectable for tests and compatible proxies.
func New(key, endpoint string, timeout time.Duration, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Adapter{
		key: key, endpoint: endpoint,
		client: httpx.NewVendorClient(string(vocab.VendorOpenAI), timeout, logger),
	}
}

var _ ports.ImageGen = (*Adapter)(nil)

// Generate submits one image request and returns a stream containing the
// completed image, or each partial image when the request asks for partials.
func (a *Adapter) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageStream, error) {
	body, err := buildRequest(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := httpx.ContextRequest(ctx, http.MethodPost, a.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, callError(vocab.ErrBadOutput, false, err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+a.key)
	httpReq.Header.Set("Content-Type", "application/json")
	response, _, err := a.client.Do(httpReq)
	if err != nil {
		return nil, classifyTransport(ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, statusError(response)
	}
	if req.Partials > 0 {
		events, err := parseSSE(response.Body)
		if err != nil {
			return nil, callError(vocab.ErrBadOutput, false, err)
		}
		return &stream{events: events}, nil
	}
	events, err := parseResponse(response.Body)
	if err != nil {
		return nil, callError(vocab.ErrBadOutput, false, err)
	}
	return &stream{events: events}, nil
}

func buildRequest(req ports.ImageRequest) ([]byte, error) {
	if strings.TrimSpace(req.Prompt) == "" {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("image prompt is empty"))
	}
	size := req.Size
	if size == "" {
		size = "1024x1024"
	}
	payload := map[string]any{
		"model": model, "prompt": req.Prompt, "size": size,
		"background": background(req.Transparent), "output_format": "png",
	}
	if req.Partials > 0 {
		payload["stream"] = true
		payload["partial_images"] = req.Partials
	}
	return json.Marshal(payload)
}

func background(transparent bool) string {
	if transparent {
		return "transparent"
	}
	return "opaque"
}

type response struct {
	Data []struct {
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

func parseResponse(reader io.Reader) ([]ports.ImageEvent, error) {
	var value response
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return nil, fmt.Errorf("decode image response: %w", err)
	}
	if len(value.Data) == 0 {
		return nil, errors.New("image response has no data")
	}
	events := make([]ports.ImageEvent, 0, len(value.Data))
	for index, image := range value.Data {
		png, err := decodePNG(image.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode image %d: %w", index, err)
		}
		events = append(events, ports.ImageEvent{PNG: png, Index: index})
	}
	return events, nil
}

func parseSSE(reader io.Reader) ([]ports.ImageEvent, error) {
	scanner := bufio.NewScanner(reader)
	events := make([]ports.ImageEvent, 0)
	index := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" || data == "" {
			continue
		}
		var value struct {
			B64JSON string `json:"b64_json"`
		}
		if err := json.Unmarshal([]byte(data), &value); err != nil {
			return nil, fmt.Errorf("decode image event: %w", err)
		}
		if value.B64JSON == "" {
			continue
		}
		png, err := decodePNG(value.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("decode image event %d: %w", index, err)
		}
		events = append(events, ports.ImageEvent{PNG: png, Partial: true, Index: index})
		index++
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read image events: %w", err)
	}
	if len(events) == 0 {
		return nil, errors.New("image stream has no images")
	}
	return events, nil
}

func decodePNG(encoded string) ([]byte, error) {
	if encoded == "" {
		return nil, errors.New("image has empty b64_json")
	}
	value, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode b64_json: %w", err)
	}
	if len(value) < 8 || !bytes.Equal(value[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil, errors.New("image is not a PNG")
	}
	return value, nil
}

type stream struct {
	events []ports.ImageEvent
	closed bool
}

func (s *stream) Recv() (ports.ImageEvent, error) {
	if s.closed || len(s.events) == 0 {
		return ports.ImageEvent{}, io.EOF
	}
	event := s.events[0]
	s.events = s.events[1:]
	return event, nil
}

func (s *stream) Close() error {
	s.closed = true
	s.events = nil
	return nil
}

func statusError(response *http.Response) error {
	message, _ := io.ReadAll(response.Body)
	kind, retryable := vocab.ErrUnavailable, response.StatusCode >= 500
	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		kind = vocab.ErrAuth
	case http.StatusTooManyRequests:
		kind, retryable = vocab.ErrRateLimited, true
	case http.StatusBadRequest:
		kind = vocab.ErrBadOutput
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
	return &ports.CallError{Vendor: vocab.VendorOpenAI, Kind: kind, Retryable: retryable, Err: err}
}
