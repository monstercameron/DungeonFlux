package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type fakeLLM struct {
	jsonFn   func(context.Context) (json.RawMessage, error)
	streamFn func(context.Context) (ports.TextStream, error)
}

func (f fakeLLM) JSON(ctx context.Context, _ ports.TextRequest, _ ports.Schema) (json.RawMessage, error) {
	return f.jsonFn(ctx)
}

func (f fakeLLM) StreamText(ctx context.Context, _ ports.TextRequest) (ports.TextStream, error) {
	return f.streamFn(ctx)
}

type fakeStream struct {
	items  chan string
	closed chan struct{}
	once   sync.Once
}

func newFakeStream(values ...string) *fakeStream {
	items := make(chan string, len(values))
	for _, value := range values {
		items <- value
	}
	close(items)
	return &fakeStream{items: items, closed: make(chan struct{})}
}

func (s *fakeStream) Recv() (string, error) {
	select {
	case value, ok := <-s.items:
		if !ok {
			return "", io.EOF
		}
		return value, nil
	case <-s.closed:
		return "", context.Canceled
	}
}

func (s *fakeStream) Close() error {
	s.once.Do(func() { close(s.closed) })
	return nil
}

func TestChain_JSONReturnsFirstAndCancelsLoser(t *testing.T) {
	started := make(chan struct{})
	loserCanceled := make(chan struct{})
	first := fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) {
		return json.RawMessage(`{"winner":true}`), nil
	}, streamFn: func(context.Context) (ports.TextStream, error) { return newFakeStream(), nil }}
	second := fakeLLM{jsonFn: func(ctx context.Context) (json.RawMessage, error) {
		close(started)
		<-ctx.Done()
		close(loserCanceled)
		return nil, ctx.Err()
	}, streamFn: func(context.Context) (ports.TextStream, error) { return newFakeStream(), nil }}
	chain := New([]ports.LLM{first, second}, Config{})
	got, err := chain.JSON(context.Background(), ports.TextRequest{}, ports.Schema{})
	if err != nil || string(got) != `{"winner":true}` {
		t.Fatalf("JSON = %s, %v", got, err)
	}
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("loser did not start")
	}
	select {
	case <-loserCanceled:
	case <-time.After(time.Second):
		t.Fatal("loser was not canceled")
	}
}

func TestChain_JSONDeadlineReturnsError(t *testing.T) {
	// A context deadline remains observable even when every link is canceled.
	link := fakeLLM{jsonFn: func(ctx context.Context) (json.RawMessage, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}, streamFn: func(context.Context) (ports.TextStream, error) { return newFakeStream(), nil }}
	chain := New([]ports.LLM{link}, Config{Deadline: time.Millisecond})
	_, err := chain.JSON(context.Background(), ports.TextRequest{}, ports.Schema{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v", err)
	}
}

func TestChain_StreamTextFallbackWinsAndClosesPrimary(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		primary := newBlockingStream()
		fallback := newFakeStream("fallback")
		chain := New([]ports.LLM{
			fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return nil, errors.New("unused") }, streamFn: func(context.Context) (ports.TextStream, error) { return primary, nil }},
			fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return nil, errors.New("unused") }, streamFn: func(context.Context) (ports.TextStream, error) { return fallback, nil }},
		}, Config{HedgeDelay: time.Millisecond})
		stream, err := chain.StreamText(context.Background(), ports.TextRequest{})
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		text, err := stream.Recv()
		if err != nil || text != "fallback" {
			t.Fatalf("Recv = %q, %v", text, err)
		}
		synctest.Wait()
		select {
		case <-primary.closed:
		default:
			t.Fatal("primary was not closed")
		}
	})
}

func TestDeadline_StreamCloseCancelsContext(t *testing.T) {
	started := make(chan struct{})
	link := fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return nil, nil }, streamFn: func(ctx context.Context) (ports.TextStream, error) {
		close(started)
		return newBlockingStreamWithContext(ctx), nil
	}}
	stream, err := Deadline(link, time.Second).StreamText(context.Background(), ports.TextRequest{})
	if err != nil {
		t.Fatal(err)
	}
	<-started
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

type blockingStream struct {
	closed chan struct{}
	once   sync.Once
}

func newBlockingStream() *blockingStream { return &blockingStream{closed: make(chan struct{})} }

func newBlockingStreamWithContext(ctx context.Context) *blockingStream {
	s := newBlockingStream()
	go func() {
		<-ctx.Done()
		_ = s.Close()
	}()
	return s
}

func (s *blockingStream) Recv() (string, error) {
	<-s.closed
	return "", context.Canceled
}

func (s *blockingStream) Close() error {
	s.once.Do(func() { close(s.closed) })
	return nil
}
