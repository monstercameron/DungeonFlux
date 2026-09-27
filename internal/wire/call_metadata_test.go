package wire

import (
	"context"
	"encoding/json"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/config"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type positionedRecordingStore map[ports.RecKey]domain.Recording

func (s positionedRecordingStore) Get(_ context.Context, key ports.RecKey) (domain.Recording, bool, error) {
	v, ok := s[key]
	return v, ok, nil
}
func (s positionedRecordingStore) Put(_ context.Context, key ports.RecKey, value domain.Recording) error {
	s[key] = value
	return nil
}

type positionSpy struct{ metas []ports.CallMeta }

func (s *positionSpy) JSON(_ context.Context, req ports.TextRequest, _ ports.Schema) (json.RawMessage, error) {
	s.metas = append(s.metas, req.Meta)
	return json.RawMessage(`{"line":"recorded"}`), nil
}

func (s *positionSpy) StreamText(_ context.Context, req ports.TextRequest) (ports.TextStream, error) {
	s.metas = append(s.metas, req.Meta)
	return safeStream{}, nil
}

func TestSequenceLLM_ResolvesRuntimePositionBeforeRecordingAndReplay(t *testing.T) {
	store := positionedRecordingStore{}
	live := &positionSpy{}
	llm := sequenceLLM(config.Config{}, live, store)
	req := ports.TextRequest{Meta: ports.CallMeta{Role: vocab.RoleNPCReply, Locale: "es"}}
	meta := ports.CallMeta{Run: "run", Phase: vocab.StateConversation, Seat: 2, Index: 7}
	ctx := ports.WithCallMeta(context.Background(), meta)
	if _, err := llm.JSON(ctx, req, ports.Schema{}); err != nil {
		t.Fatal(err)
	}
	stream, err := llm.StreamText(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	for {
		_, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	_ = stream.Close()
	for _, got := range live.metas {
		if got.Phase != meta.Phase || got.Index != 7 || got.Seat != 2 || got.Locale != "es" || got.Role != vocab.RoleNPCReply {
			t.Fatalf("live metadata=%+v", got)
		}
	}
	safe := config.Config{}
	safe.Features.SequenceMode = true
	replay := sequenceLLM(safe, failingLLM{}, store)
	if value, err := replay.JSON(ctx, req, ports.Schema{}); err != nil || string(value) != `{"line":"recorded"}` {
		t.Fatalf("positioned replay=%s,%v", value, err)
	}
	meta.Index++
	if _, err := replay.JSON(ports.WithCallMeta(ctx, meta), req, ports.Schema{}); err == nil {
		t.Fatal("next position reused the previous reply")
	}
	if _, err := sequenceLLM(config.Config{}, live, nil).JSON(ctx, req, ports.Schema{}); err != nil {
		t.Fatal(err)
	}
	if sequenceLLM(config.Config{}, nil, nil) != nil {
		t.Fatal("nil provider became callable")
	}
}
