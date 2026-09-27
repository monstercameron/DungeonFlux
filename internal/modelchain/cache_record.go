package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// RecordReplay records successful calls and serves the matching recording
// when ForceReplay is set or the live link fails.
func RecordReplay(next ports.LLM, store ports.Recordings, adapter string) ports.LLM {
	return recordedLLM{next: next, store: store, adapter: adapter}
}

type recordedLLM struct {
	next    ports.LLM
	store   ports.Recordings
	adapter string
}

func (r recordedLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := recordingKey(r.adapter, req, "json", schema)
	if err != nil {
		return nil, err
	}
	if req.Meta.ForceReplay {
		return r.readJSON(ctx, key)
	}
	value, err := r.next.JSON(ctx, req, schema)
	if canceled := cancellationError(ctx, err); canceled != nil {
		return nil, canceled
	}
	if err == nil {
		if putErr := r.store.Put(ctx, key, domain.Recording{Text: string(value)}); putErr != nil {
			return nil, putErr
		}
		return value, nil
	}
	if replay, replayErr := r.readJSON(ctx, key); replayErr == nil {
		return replay, nil
	}
	return nil, err
}

func (r recordedLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	key, err := recordingKey(r.adapter, req, "text", ports.Schema{})
	if err != nil {
		return nil, err
	}
	if req.Meta.ForceReplay {
		value, err := r.readText(ctx, key)
		if err != nil {
			return nil, err
		}
		return newReplayStream(ctx, value), nil
	}
	stream, err := r.next.StreamText(ctx, req)
	if canceled := cancellationError(ctx, err); canceled != nil {
		if stream != nil {
			_ = stream.Close()
		}
		return nil, canceled
	}
	if err != nil {
		if value, replayErr := r.readText(ctx, key); replayErr == nil {
			return newReplayStream(ctx, value), nil
		}
		return nil, err
	}
	return newStoredStream(ctx, stream, func(ctx context.Context, value []byte) error {
		return r.store.Put(ctx, key, domain.Recording{Text: string(value)})
	}), nil
}

func (r recordedLLM) readJSON(ctx context.Context, key ports.RecKey) (json.RawMessage, error) {
	recording, ok, err := r.store.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if !ok || recording.Text == "" {
		return nil, errors.New("modelchain: recording not found")
	}
	return json.RawMessage(recording.Text), nil
}

func (r recordedLLM) readText(ctx context.Context, key ports.RecKey) (string, error) {
	recording, ok, err := r.store.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", errors.New("modelchain: recording not found")
	}
	return recording.Text, nil
}

func newReplayStream(ctx context.Context, text string) ports.TextStream {
	return &replayStream{ctx: ctx, chunks: []string{text}}
}

type replayStream struct {
	mu     sync.Mutex
	ctx    context.Context
	chunks []string
	closed bool
}

func (s *replayStream) Recv() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.ctx.Err(); err != nil {
		return "", err
	}
	if s.closed || len(s.chunks) == 0 {
		return "", io.EOF
	}
	text := s.chunks[0]
	s.chunks = s.chunks[1:]
	return text, nil
}

func (s *replayStream) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

var _ ports.LLM = recordedLLM{}
