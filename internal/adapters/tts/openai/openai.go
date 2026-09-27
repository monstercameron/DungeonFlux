package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/httpx"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const (
	defaultEndpoint = "https://api.openai.com/v1/"
	model           = openai.SpeechModelGPT4oMiniTTS
	outputRate      = 24000
)

// Adapter implements ports.TTS with OpenAI's PCM speech endpoint.
type Adapter struct {
	client *openai.Client
}

// New constructs an OpenAI TTS adapter. Endpoint is the API base URL and is
// injectable for tests; an empty endpoint uses the public OpenAI API.
func New(key, endpoint string, timeout time.Duration, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}
	httpClient := httpx.NewVendorClient(string(vocab.VendorOpenAI), timeout, logger)
	client := openai.NewClient(
		option.WithAPIKey(key),
		option.WithBaseURL(endpoint),
		option.WithHTTPClient(httpClient.HTTPClient()),
		option.WithMaxRetries(0),
	)
	return &Adapter{client: &client}
}

var _ ports.TTS = (*Adapter)(nil)

// Stream renders all text received from text and returns it as one PCM chunk.
// OpenAI's PCM response is signed 16-bit little-endian mono audio at 24 kHz.
func (a *Adapter) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	input, err := collectText(text)
	if err != nil {
		return nil, callError(classifyContext(ctx), false, err)
	}
	if strings.TrimSpace(input) == "" {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("tts text is empty"))
	}
	if strings.TrimSpace(req.VoiceID) == "" {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("tts voice is empty"))
	}
	response, err := a.client.Audio.Speech.New(ctx, openai.AudioSpeechNewParams{
		Input:          input,
		Model:          model,
		Voice:          openai.AudioSpeechNewParamsVoiceUnion{OfString: openai.String(req.VoiceID)},
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormatPCM,
	})
	if err != nil {
		return nil, callError(classifyContext(ctx), retryable(ctx, err), err)
	}
	defer response.Body.Close()
	pcm, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, callError(classifyContext(ctx), true, err)
	}
	if len(pcm) == 0 {
		return nil, callError(vocab.ErrBadOutput, false, errors.New("tts response is empty"))
	}
	return &stream{chunk: ports.PCMChunk{SampleRate: outputRate, S16LE: pcm}}, nil
}

func collectText(text ports.TextStream) (string, error) {
	if text == nil {
		return "", errors.New("tts text stream is nil")
	}
	defer text.Close()
	var builder strings.Builder
	for {
		part, err := text.Recv()
		if err == io.EOF {
			return builder.String(), nil
		}
		if err != nil {
			return "", fmt.Errorf("read tts text: %w", err)
		}
		builder.WriteString(part)
	}
}

type stream struct {
	chunk  ports.PCMChunk
	closed bool
}

func (s *stream) Recv() (ports.PCMChunk, error) {
	if s.closed || len(s.chunk.S16LE) == 0 {
		return ports.PCMChunk{}, io.EOF
	}
	chunk := s.chunk
	s.chunk.S16LE = nil
	return chunk, nil
}

func (s *stream) Close() error {
	s.closed = true
	s.chunk = ports.PCMChunk{}
	return nil
}

func classifyContext(ctx context.Context) vocab.ErrKind {
	if errors.Is(ctx.Err(), context.Canceled) {
		return vocab.ErrCanceled
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return vocab.ErrTimeout
	}
	return vocab.ErrUnavailable
}

func retryable(ctx context.Context, err error) bool {
	return ctx.Err() == nil && err != nil
}

func callError(kind vocab.ErrKind, retryable bool, err error) error {
	return &ports.CallError{Vendor: vocab.VendorOpenAI, Kind: kind, Retryable: retryable, Err: err}
}
