package elevenlabs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const (
	defaultEndpoint = "wss://api.elevenlabs.io/v1/text-to-speech"
	modelID         = "eleven_flash_v2_5"
	sampleRate      = 24000
	idleTimeout     = 20 * time.Second
	outputBuffer    = 16
)

// Adapter streams live PCM audio from ElevenLabs.
type Adapter struct {
	key      string
	endpoint string
	logger   *slog.Logger
	dialer   *websocket.Dialer
	clock    clock.Clock
}

// New returns an ElevenLabs adapter. Endpoint may be overridden for tests;
// an empty endpoint uses the ElevenLabs production endpoint.
func New(key, endpoint string, logger *slog.Logger) *Adapter {
	if endpoint == "" {
		endpoint = defaultEndpoint
	}
	return &Adapter{key: key, endpoint: strings.TrimRight(endpoint, "/"), logger: logger, dialer: websocket.DefaultDialer, clock: clock.Real{}}
}

var _ ports.TTS = (*Adapter)(nil)

// Stream opens a stream-input connection and starts its reader and writer.
func (a *Adapter) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	if text == nil {
		return nil, &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: vocab.ErrBadOutput, Err: errors.New("text stream is required")}
	}
	if req.VoiceID == "" {
		return nil, &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: vocab.ErrBadOutput, Err: errors.New("voice ID is required")}
	}
	wsURL, err := streamURL(a.endpoint, req.VoiceID)
	if err != nil {
		return nil, &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: vocab.ErrBadOutput, Err: err}
	}
	header := http.Header{"xi-api-key": []string{a.key}}
	conn, response, err := a.dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return nil, &ports.CallError{Vendor: vocab.VendorElevenLabs, Kind: vocab.ErrUnavailable, Retryable: true, Err: err}
	}
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	stream := newPCMStream(conn, text, ctx, a.logger, wsURL, a.clock)
	return stream, nil
}

func streamURL(endpoint, voiceID string) (string, error) {
	base, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("parse ElevenLabs endpoint: %w", err)
	}
	if base.Scheme != "ws" && base.Scheme != "wss" {
		return "", fmt.Errorf("ElevenLabs endpoint must use ws or wss")
	}
	path := strings.TrimRight(base.Path, "/") + "/" + voiceID + "/stream-input"
	base.Path = path
	base.RawPath = strings.TrimRight(base.EscapedPath(), "/") + "/" + url.PathEscape(voiceID) + "/stream-input"
	query := base.Query()
	query.Set("model_id", modelID)
	query.Set("output_format", "pcm_24000")
	query.Set("auto_mode", "true")
	base.RawQuery = query.Encode()
	return base.String(), nil
}

type pcmStream struct {
	conn    *websocket.Conn
	text    ports.TextStream
	ctx     context.Context
	logger  *slog.Logger
	url     string
	clock   clock.Clock
	frames  chan pcmResult
	done    chan struct{}
	once    sync.Once
	workers sync.WaitGroup
}

type pcmResult struct {
	chunk ports.PCMChunk
	err   error
}

func newPCMStream(conn *websocket.Conn, text ports.TextStream, ctx context.Context, logger *slog.Logger, wsURL string, timer clock.Clock) *pcmStream {
	s := &pcmStream{conn: conn, text: text, ctx: ctx, logger: logger, url: wsURL, clock: timer, frames: make(chan pcmResult, outputBuffer), done: make(chan struct{})}
	s.workers.Add(2)
	go func() { defer s.workers.Done(); s.readLoop() }()
	go func() { defer s.workers.Done(); s.writeLoop() }()
	go func() { s.workers.Wait(); close(s.frames) }()
	go func() {
		select {
		case <-ctx.Done():
			s.stop(ctx.Err())
		case <-s.done:
		}
	}()
	return s
}

// Recv returns the next decoded PCM frame, or io.EOF after the final frame.
func (s *pcmStream) Recv() (ports.PCMChunk, error) {
	result, ok := <-s.frames
	if !ok {
		return ports.PCMChunk{}, io.EOF
	}
	return result.chunk, result.err
}

// Close stops both stream workers and closes the WebSocket.
func (s *pcmStream) Close() error {
	s.stop(nil)
	return nil
}

func (s *pcmStream) readLoop() {
	for {
		_ = s.conn.SetReadDeadline(s.clock.Now().Add(idleTimeout))
		_, payload, err := s.conn.ReadMessage()
		if err != nil {
			if !s.stopped() && !errors.Is(err, websocket.ErrCloseSent) {
				s.emitError(fmt.Errorf("read ElevenLabs stream: %w", err))
			}
			s.stop(err)
			return
		}
		message, err := parseMessage(payload)
		if err != nil {
			s.emitError(err)
			s.stop(err)
			return
		}
		if len(message.Audio) > 0 {
			select {
			case s.frames <- pcmResult{chunk: ports.PCMChunk{SampleRate: sampleRate, S16LE: message.Audio}}:
			case <-s.done:
				return
			}
		}
		if message.IsFinal {
			s.stop(nil)
			return
		}
	}
}

func (s *pcmStream) writeLoop() {
	if err := s.writeJSON(startMessage()); err != nil {
		s.emitError(err)
		s.stop(err)
		return
	}
	for {
		text, err := s.text.Recv()
		if errors.Is(err, io.EOF) {
			err = s.writeJSON(textMessage(""))
			if err != nil {
				s.emitError(err)
				s.stop(err)
			}
			return
		}
		if err != nil {
			s.emitError(fmt.Errorf("read TTS text: %w", err))
			s.stop(err)
			return
		}
		if text == "" {
			continue
		}
		if err := s.writeJSON(textMessage(text)); err != nil {
			s.emitError(err)
			s.stop(err)
			return
		}
	}
}

func (s *pcmStream) writeJSON(value any) error {
	select {
	case <-s.done:
		return context.Canceled
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
	}
	if err := s.conn.SetWriteDeadline(s.clock.Now().Add(idleTimeout)); err != nil {
		return fmt.Errorf("set ElevenLabs write deadline: %w", err)
	}
	if err := s.conn.WriteJSON(value); err != nil {
		return fmt.Errorf("write ElevenLabs stream: %w", err)
	}
	return nil
}

func (s *pcmStream) emitError(err error) {
	select {
	case s.frames <- pcmResult{err: err}:
	case <-s.done:
	}
}

func (s *pcmStream) stop(err error) {
	s.once.Do(func() {
		close(s.done)
		_ = s.text.Close()
		_ = s.conn.Close()
		if s.logger != nil {
			attrs := []any{"vendor", string(vocab.VendorElevenLabs), "url", s.url, "status", 101}
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, websocket.ErrCloseSent) {
				attrs = append(attrs, "err", err)
			}
			s.logger.Info("call", attrs...)
		}
	})
}

func (s *pcmStream) stopped() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

type streamMessage struct {
	Audio   []byte
	IsFinal bool
}

func parseMessage(payload []byte) (streamMessage, error) {
	var wire struct {
		Audio   string `json:"audio"`
		IsFinal bool   `json:"isFinal"`
		Error   string `json:"error"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return streamMessage{}, fmt.Errorf("decode ElevenLabs message: %w", err)
	}
	if wire.Error != "" {
		return streamMessage{}, fmt.Errorf("ElevenLabs stream error: %s", wire.Error)
	}
	audio, err := base64.StdEncoding.DecodeString(wire.Audio)
	if err != nil {
		return streamMessage{}, fmt.Errorf("decode ElevenLabs PCM: %w", err)
	}
	return streamMessage{Audio: audio, IsFinal: wire.IsFinal}, nil
}

func startMessage() map[string]any {
	return map[string]any{"text": " ", "xi_api_key": "", "generation_config": map[string]any{"chunk_length_schedule": []int{50, 90, 120}}}
}

func textMessage(text string) map[string]string { return map[string]string{"text": text} }
