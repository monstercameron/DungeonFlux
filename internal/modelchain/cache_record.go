package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"

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
	key, err := recordingKey(r.adapter, req, "json", schema)
	if err != nil {
		return nil, err
	}
	if req.Meta.ForceReplay {
		return r.readJSON(ctx, key)
	}
	value, err := r.next.JSON(ctx, req, schema)
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
	key, err := recordingKey(r.adapter, req, "text", ports.Schema{})
	if err != nil {
		return nil, err
	}
	if req.Meta.ForceReplay {
		value, err := r.readText(ctx, key)
		if err != nil {
			return nil, err
		}
		return newReplayStream(value), nil
	}
	stream, err := r.next.StreamText(ctx, req)
	if err != nil {
		if value, replayErr := r.readText(ctx, key); replayErr == nil {
			return newReplayStream(value), nil
		}
		return nil, err
	}
	return &recordStream{source: stream, store: r.store, key: key}, nil
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

type recordStream struct {
	source ports.TextStream
	store  ports.Recordings
	key    ports.RecKey
	text   []byte
	done   bool
}

func (s *recordStream) Recv() (string, error) {
	text, err := s.source.Recv()
	if err == nil {
		s.text = append(s.text, text...)
		return text, nil
	}
	if errors.Is(err, io.EOF) && !s.done {
		s.done = true
		if putErr := s.store.Put(context.Background(), s.key, domain.Recording{Text: string(s.text)}); putErr != nil {
			return "", putErr
		}
	}
	return "", err
}

func (s *recordStream) Close() error {
	s.done = true
	return s.source.Close()
}

func newReplayStream(text string) ports.TextStream {
	return &replayStream{chunks: []string{text}}
}

type replayStream struct {
	chunks []string
	closed bool
}

func (s *replayStream) Recv() (string, error) {
	if s.closed || len(s.chunks) == 0 {
		return "", io.EOF
	}
	text := s.chunks[0]
	s.chunks = s.chunks[1:]
	return text, nil
}

func (s *replayStream) Close() error {
	s.closed = true
	return nil
}

var _ ports.LLM = recordedLLM{}
var _ ports.TextStream = (*recordStream)(nil)
