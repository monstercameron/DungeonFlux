package schemaflux

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	schemaflux "github.com/monstercameron/schemaflux"
)

// Config configures one SchemaFlux provider link.
type Config struct {
	// Provider is "openai" (Luna), "cerebras", or "local".
	Provider string
	APIKey   string
	BaseURL  string
	Model    string
	// ReasoningEffort is sent in the vendor-specific request field.
	ReasoningEffort string
	Timeout         time.Duration
	HTTPClient      *http.Client
}

// Adapter implements ports.LLM with one SchemaFlux provider instance.
type Adapter struct {
	provider schemaflux.Provider
	model    string
	vendor   vocab.VendorName
}

var _ ports.LLM = (*Adapter)(nil)

// New constructs an adapter without using SchemaFlux's global registry.
func New(cfg Config) (*Adapter, error) {
	name := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if name == "" {
		name = "openai"
	}
	if cfg.ReasoningEffort == "" {
		cfg.ReasoningEffort = defaultEffort(name)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: cfg.Timeout}
	}
	client = withReasoning(client, cfg.ReasoningEffort)
	pc := schemaflux.ProviderConfig{
		APIKey: cfg.APIKey, BaseURL: cfg.BaseURL, Timeout: cfg.Timeout,
		HTTPClient: client,
	}
	var (
		provider schemaflux.Provider
		err      error
	)
	switch name {
	case "openai", "luna":
		provider, err = schemaflux.NewOpenAIProvider(pc)
		name = "openai"
	case "cerebras", "qwen":
		provider, err = schemaflux.NewCerebrasProvider(pc)
		name = "cerebras"
	case "local", "llama":
		provider, err = schemaflux.NewOpenAICompatibleProvider("local", pc)
		name = "local"
	default:
		return nil, fmt.Errorf("schemaflux: unsupported provider %q", cfg.Provider)
	}
	if err != nil {
		return nil, fmt.Errorf("schemaflux: configure %s: %w", name, err)
	}
	return &Adapter{provider: provider, model: cfg.Model, vendor: vendor(name)}, nil
}

// NewOpenAI constructs a Luna/Responses adapter.
func NewOpenAI(apiKey, baseURL, model, effort string, client *http.Client) (*Adapter, error) {
	return New(Config{Provider: "openai", APIKey: apiKey, BaseURL: baseURL, Model: model, ReasoningEffort: effort, HTTPClient: client})
}

// NewCerebras constructs a Cerebras chat-completions adapter.
func NewCerebras(apiKey, baseURL, model string, client *http.Client) (*Adapter, error) {
	return New(Config{Provider: "cerebras", APIKey: apiKey, BaseURL: baseURL, Model: model, ReasoningEffort: "none", HTTPClient: client})
}

// JSON sends a strict-schema JSON completion.
func (a *Adapter) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	request, err := a.request(req, "json", schema)
	if err != nil {
		return nil, err
	}
	response, err := a.provider.Complete(ctx, request)
	if err != nil {
		return nil, a.callError(ctx, err)
	}
	if strings.TrimSpace(response.Content) == "" {
		return nil, a.callError(ctx, errors.New("provider returned empty output"))
	}
	return json.RawMessage(response.Content), nil
}

// StreamText starts a provider text stream. The returned stream is pull-based
// and closes the provider iterator when its context is canceled.
func (a *Adapter) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	request, err := a.request(req, "text", ports.Schema{})
	if err != nil {
		return nil, err
	}
	method := reflect.ValueOf(a.provider).MethodByName("CompleteStream")
	if !method.IsValid() {
		return nil, &ports.CallError{Vendor: a.vendor, Kind: vocab.ErrUnavailable, Err: errors.New("provider does not support streaming")}
	}
	streamCtx, cancel := context.WithCancel(ctx)
	return newStream(streamCtx, cancel, method, request, a.vendor), nil
}

func (a *Adapter) request(req ports.TextRequest, format string, schema ports.Schema) (schemaflux.CompletionRequest, error) {
	system, user := prompts(req.Messages)
	result := schemaflux.CompletionRequest{Model: a.model, SystemPrompt: system, UserPrompt: user, MaxTokens: req.MaxTokens, ResponseFormat: format}
	if format != "json" {
		return result, nil
	}
	if len(schema.JSON) == 0 || strings.TrimSpace(schema.Name) == "" {
		return result, &ports.CallError{Vendor: a.vendor, Kind: vocab.ErrBadOutput, Err: errors.New("JSON schema and name are required")}
	}
	var value map[string]any
	if err := json.Unmarshal(schema.JSON, &value); err != nil {
		return result, &ports.CallError{Vendor: a.vendor, Kind: vocab.ErrBadOutput, Err: fmt.Errorf("decode schema: %w", err)}
	}
	result.SchemaName, result.JSONSchema = schema.Name, value
	return result, nil
}

func prompts(messages []ports.Message) (string, string) {
	var system, user strings.Builder
	for _, message := range messages {
		if message.Role == vocab.MsgSystem {
			system.WriteString(message.Text)
			continue
		}
		if user.Len() > 0 {
			user.WriteString("\n")
		}
		user.WriteString(message.Text)
	}
	return system.String(), user.String()
}

func (a *Adapter) callError(ctx context.Context, err error) error {
	kind := vocab.ErrUnavailable
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		kind = vocab.ErrCanceled
	}
	var api *schemaflux.APIError
	var rate *schemaflux.RateLimitError
	if errors.As(err, &rate) {
		kind = vocab.ErrRateLimited
	} else if errors.As(err, &api) {
		kind = vocab.ErrBadOutput
		if api.StatusCode == http.StatusUnauthorized || api.StatusCode == http.StatusForbidden {
			kind = vocab.ErrAuth
		}
	}
	return &ports.CallError{Vendor: a.vendor, Kind: kind, Retryable: kind == vocab.ErrUnavailable || kind == vocab.ErrRateLimited, Err: err}
}

func defaultEffort(provider string) string {
	if provider == "cerebras" || provider == "qwen" {
		return "none"
	}
	return "none"
}

func vendor(name string) vocab.VendorName {
	if name == "cerebras" {
		return vocab.VendorCerebras
	}
	if name == "local" {
		return vocab.VendorLocal
	}
	return vocab.VendorOpenAI
}

type reasoningTransport struct {
	base   http.RoundTripper
	effort string
}

func withReasoning(client *http.Client, effort string) *http.Client {
	if strings.TrimSpace(effort) == "" {
		return client
	}
	copy := *client
	base := copy.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	copy.Transport = &reasoningTransport{base: base, effort: effort}
	return &copy
}

func (t *reasoningTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body == nil {
		return t.base.RoundTrip(req)
	}
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()
	var body map[string]any
	if json.Unmarshal(raw, &body) == nil {
		if strings.HasSuffix(req.URL.Path, "/responses") {
			body["reasoning"] = map[string]any{"effort": t.effort}
		} else if strings.HasSuffix(req.URL.Path, "/chat/completions") {
			body["reasoning_effort"] = t.effort
		}
		if encoded, marshalErr := json.Marshal(body); marshalErr == nil {
			raw = encoded
		}
	}
	req.Body = io.NopCloser(bytes.NewReader(raw))
	req.ContentLength = int64(len(raw))
	return t.base.RoundTrip(req)
}

type stream struct {
	ctx    context.Context
	cancel context.CancelFunc
	items  chan streamItem
	vendor vocab.VendorName
}

type streamItem struct {
	text string
	err  error
	done bool
}

func newStream(ctx context.Context, cancel context.CancelFunc, method reflect.Value, request schemaflux.CompletionRequest, vendor vocab.VendorName) *stream {
	s := &stream{ctx: ctx, cancel: cancel, items: make(chan streamItem, 8), vendor: vendor}
	go func() {
		defer close(s.items)
		result := method.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(request)})
		sequence := result[0]
		callbackType := sequence.Type().In(0)
		callback := reflect.MakeFunc(callbackType, func(args []reflect.Value) []reflect.Value {
			chunk, callErr := args[0], args[1]
			item := streamItem{}
			if !callErr.IsNil() {
				item.err = callErr.Interface().(error)
			} else {
				delta := chunk.FieldByName("Delta")
				done := chunk.FieldByName("Done")
				item.text, item.done = delta.String(), done.Bool()
			}
			select {
			case s.items <- item:
				return []reflect.Value{reflect.ValueOf(true)}
			case <-ctx.Done():
				return []reflect.Value{reflect.ValueOf(false)}
			}
		})
		sequence.Call([]reflect.Value{callback})
	}()
	return s
}

func (s *stream) Recv() (string, error) {
	select {
	case <-s.ctx.Done():
		return "", s.ctx.Err()
	case item, ok := <-s.items:
		if !ok {
			return "", io.EOF
		}
		if item.err != nil {
			return "", (&Adapter{vendor: s.vendor}).callError(s.ctx, item.err)
		}
		if item.done {
			return "", io.EOF
		}
		return item.text, nil
	}
}

func (s *stream) Close() error {
	s.cancel()
	return nil
}
