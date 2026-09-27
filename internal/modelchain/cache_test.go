package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type memoryCache struct{ values map[string][]byte }

func (m *memoryCache) Get(_ context.Context, adapter, key string) ([]byte, bool, error) {
	value, ok := m.values[adapter+key]
	return value, ok, nil
}
func (m *memoryCache) Put(_ context.Context, adapter, key string, value []byte) error {
	m.values[adapter+key] = append([]byte(nil), value...)
	return nil
}

type memoryRecordings struct {
	values map[ports.RecKey]domain.Recording
}

func (m *memoryRecordings) Get(_ context.Context, key ports.RecKey) (domain.Recording, bool, error) {
	value, ok := m.values[key]
	return value, ok, nil
}
func (m *memoryRecordings) Put(_ context.Context, key ports.RecKey, value domain.Recording) error {
	m.values[key] = value
	return nil
}

func TestCached_JSONAndTextHit(t *testing.T) {
	calls := 0
	link := fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		calls++
		return json.RawMessage(`{"ok":true}`), nil
	}, streamFn: func(context.Context) (ports.TextStream, error) {
		calls++
		return newFakeStream("hello", " world"), nil
	}}
	store := &memoryCache{values: map[string][]byte{}}
	decorator := Cached(link, store, "test")
	for range 2 {
		value, err := decorator.JSON(context.Background(), ports.TextRequest{}, ports.Schema{Name: "x"})
		if err != nil || string(value) != `{"ok":true}` {
			t.Fatalf("JSON = %s, %v", value, err)
		}
	}
	stream, err := decorator.StreamText(context.Background(), ports.TextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("stream end = %v", err)
	}
	stream, err = decorator.StreamText(context.Background(), ports.TextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var text string
	for {
		part, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			t.Fatal(recvErr)
		}
		text += part
	}
	if text != "hello world" || calls != 2 {
		t.Fatalf("text=%q calls=%d", text, calls)
	}
}

func TestRecordReplay_JSONFallbackAndForcedText(t *testing.T) {
	link := fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		return json.RawMessage(`{"value":7}`), nil
	}, streamFn: func(context.Context) (ports.TextStream, error) { return newFakeStream("recorded"), nil }}
	store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
	decorator := RecordReplay(link, store, "luna")
	req := ports.TextRequest{Meta: ports.CallMeta{Index: 3}}
	value, err := decorator.JSON(context.Background(), req, ports.Schema{})
	if err != nil || string(value) != `{"value":7}` {
		t.Fatalf("JSON = %s, %v", value, err)
	}
	req.Meta.ForceReplay = true
	value, err = decorator.JSON(context.Background(), req, ports.Schema{})
	if err != nil || string(value) != `{"value":7}` {
		t.Fatalf("replay JSON = %s, %v", value, err)
	}
	stream, err := decorator.StreamText(context.Background(), ports.TextRequest{Meta: ports.CallMeta{Index: 4}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) {
		t.Fatalf("record end = %v", err)
	}
	stream, err = decorator.StreamText(context.Background(), ports.TextRequest{Meta: ports.CallMeta{Index: 4, ForceReplay: true}})
	if err != nil {
		t.Fatal(err)
	}
	text, err := stream.Recv()
	if err != nil || text != "recorded" {
		t.Fatalf("replay text = %q, %v", text, err)
	}
}

func TestRecordReplay_LiveFailureUsesRecording(t *testing.T) {
	store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
	record := RecordReplay(fakeLLM{streamFn: func(context.Context) (ports.TextStream, error) {
		return newFakeStream("saved"), nil
	}}, store, "x")
	runRehearsalCall(t, record, rehearsalCall{stream: true}, "saved")
	link := fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return nil, errors.New("down") }, streamFn: func(context.Context) (ports.TextStream, error) { return nil, errors.New("down") }}
	decorator := RecordReplay(link, store, "x")
	stream, err := decorator.StreamText(context.Background(), ports.TextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	text, err := stream.Recv()
	if err != nil || text != "saved" {
		t.Fatalf("fallback = %q, %v", text, err)
	}
}
