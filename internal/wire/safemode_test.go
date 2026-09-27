package wire

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type safeRecordingStore struct{ value domain.Recording }

func (s *safeRecordingStore) Get(context.Context, ports.RecKey) (domain.Recording, bool, error) {
	if s.value.Text == "" {
		return domain.Recording{}, false, nil
	}
	return s.value, true, nil
}

func (s *safeRecordingStore) Put(_ context.Context, _ ports.RecKey, value domain.Recording) error {
	s.value = value
	return nil
}

type safeLLM struct{}

func (safeLLM) JSON(context.Context, ports.TextRequest, ports.Schema) (json.RawMessage, error) {
	return json.RawMessage(`{"ok":true}`), nil
}

func (safeLLM) StreamText(context.Context, ports.TextRequest) (ports.TextStream, error) {
	return safeStream{}, nil
}

type safeStream struct{}

func (safeStream) Recv() (string, error) { return "live", io.EOF }
func (safeStream) Close() error          { return nil }

func TestSequenceLLM_recordsAndReplays(t *testing.T) {
	store := &safeRecordingStore{}
	base := sequenceLLM(config.Config{}, safeLLM{}, store)
	value, err := base.JSON(context.Background(), ports.TextRequest{Meta: ports.CallMeta{Index: 4}}, ports.Schema{Name: "x", JSON: json.RawMessage(`{}`)})
	if err != nil || string(value) != `{"ok":true}` {
		t.Fatalf("recorded JSON = %s, %v", value, err)
	}

	safe := config.Config{}
	safe.Features.SequenceMode = true
	replay := sequenceLLM(safe, failingLLM{}, store)
	value, err = replay.JSON(context.Background(), ports.TextRequest{Meta: ports.CallMeta{Index: 4}}, ports.Schema{Name: "x", JSON: json.RawMessage(`{}`)})
	if err != nil || string(value) != `{"ok":true}` {
		t.Fatalf("replayed JSON = %s, %v", value, err)
	}
}

type failingLLM struct{}

func (failingLLM) JSON(context.Context, ports.TextRequest, ports.Schema) (json.RawMessage, error) {
	return nil, errors.New("uplink unavailable")
}

func (failingLLM) StreamText(context.Context, ports.TextRequest) (ports.TextStream, error) {
	return nil, errors.New("uplink unavailable")
}
