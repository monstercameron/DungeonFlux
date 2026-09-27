package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

type terminalTextStream struct {
	final string
	err   error
	step  int
}

func (s *terminalTextStream) Recv() (string, error) {
	s.step++
	switch s.step {
	case 1:
		return "first ", nil
	case 2:
		return s.final, s.err
	default:
		return "", io.EOF
	}
}

func (s *terminalTextStream) Close() error { return nil }

func streamDecorator(mode string, source ports.TextStream) (ports.LLM, func() []string) {
	live := fakeLLM{streamFn: func(context.Context) (ports.TextStream, error) { return source, nil }}
	if mode == "cache" {
		store := &memoryCache{values: map[string][]byte{}}
		return Cached(live, store, "test"), func() []string {
			var values []string
			for _, value := range store.values {
				values = append(values, string(value))
			}
			return values
		}
	}
	store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
	return RecordReplay(live, store, "test"), func() []string {
		var values []string
		for _, value := range store.values {
			values = append(values, value.Text)
		}
		return values
	}
}

func TestStoredDecorators_OnlyPublishCompleteUncancelledText(t *testing.T) {
	for _, mode := range []string{"cache", "recording"} {
		for _, tc := range []struct {
			name   string
			err    error
			cancel bool
			close  bool
			want   string
		}{
			{name: "terminal text", err: io.EOF, want: "first last"},
			{name: "source failure then EOF", err: io.ErrUnexpectedEOF},
			{name: "request canceled before EOF", err: io.EOF, cancel: true},
			{name: "closed before EOF", err: io.EOF, close: true},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				llm, values := streamDecorator(mode, &terminalTextStream{final: "last", err: tc.err})
				stream, err := llm.StreamText(ctx, ports.TextRequest{})
				if err != nil {
					t.Fatal(err)
				}
				defer stream.Close()
				if text, err := stream.Recv(); text != "first " || err != nil {
					t.Fatalf("first recv=%q, %v", text, err)
				}
				if tc.cancel {
					cancel()
				}
				if tc.close {
					_ = stream.Close()
				}
				text, err := stream.Recv()
				if tc.want != "" && (text != "last" || err != nil) {
					t.Fatalf("terminal recv=%q, %v", text, err)
				}
				if tc.cancel && !errors.Is(err, context.Canceled) {
					t.Fatalf("cancelled recv=%q, %v", text, err)
				}
				_, _ = stream.Recv() // An error followed by EOF must not publish a partial response.
				got := values()
				if tc.want == "" && len(got) != 0 || tc.want != "" && (len(got) != 1 || got[0] != tc.want) {
					t.Fatalf("stored=%q, want complete=%q", got, tc.want)
				}
			})
		}
	}
}

func TestStoredStream_CloseCancelsPendingPersistence(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan struct{})
		result := make(chan error, 1)
		stream := newStoredStream(context.Background(), &terminalTextStream{err: io.EOF}, func(ctx context.Context, _ []byte) error {
			close(entered)
			<-ctx.Done()
			return ctx.Err()
		})
		_, _ = stream.Recv()
		go func() { _, err := stream.Recv(); result <- err }()
		<-entered
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-result; !errors.Is(err, context.Canceled) {
			t.Fatalf("pending persistence returned %v", err)
		}
	})
}

type blockingFinalStream struct {
	entered chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func (s *blockingFinalStream) Recv() (string, error) {
	close(s.entered)
	<-s.closed
	return "late text", io.EOF
}

func (s *blockingFinalStream) Close() error { s.once.Do(func() { close(s.closed) }); return nil }

func TestStoredStream_CloseDuringReadDoesNotPublish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		source := &blockingFinalStream{entered: make(chan struct{}), closed: make(chan struct{})}
		writes := 0
		stream := newStoredStream(context.Background(), source, func(context.Context, []byte) error { writes++; return nil })
		result := make(chan error, 1)
		go func() { _, err := stream.Recv(); result <- err }()
		<-source.entered
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-result; !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
			t.Fatal(err)
		}
		if writes != 0 {
			t.Fatalf("closed stream wrote %d recordings", writes)
		}
	})
}

func TestStoredStream_ReportsStorageFailureOnce(t *testing.T) {
	failure := errors.New("disk full")
	writes := 0
	stream := newStoredStream(context.Background(), &terminalTextStream{final: "last", err: io.EOF}, func(context.Context, []byte) error { writes++; return failure })
	defer stream.Close()
	_, _ = stream.Recv()
	if text, err := stream.Recv(); text != "last" || !errors.Is(err, failure) {
		t.Fatalf("terminal=%q, %v", text, err)
	}
	if _, err := stream.Recv(); !errors.Is(err, io.EOF) || writes != 1 {
		t.Fatalf("after failure=%v writes=%d", err, writes)
	}
}

func TestRecordReplay_CancellationDoesNotUseSavedFallback(t *testing.T) {
	for _, mode := range []string{"json", "text"} {
		t.Run(mode, func(t *testing.T) {
			store := &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}
			call := rehearsalCall{stream: mode == "text"}
			recorder := RecordReplay(fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return json.RawMessage(`{"saved":true}`), nil }, streamFn: func(context.Context) (ports.TextStream, error) { return newFakeStream(`{"saved":true}`), nil }}, store, "test")
			runRehearsalCall(t, recorder, call, `{"saved":true}`)
			failed := RecordReplay(fakeLLM{jsonFn: func(context.Context) (json.RawMessage, error) { return nil, context.Canceled }, streamFn: func(context.Context) (ports.TextStream, error) { return nil, context.Canceled }}, store, "test")
			var err error
			if call.stream {
				_, err = failed.StreamText(context.Background(), call.req)
			} else {
				_, err = failed.JSON(context.Background(), call.req, call.schema)
			}
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled call replayed stale success: %v", err)
			}
		})
	}
}

func TestStoredDecorators_ReplayedStreamsRespectCallerCancellation(t *testing.T) {
	for _, mode := range []string{"cache", "recording"} {
		t.Run(mode, func(t *testing.T) {
			llm, _ := streamDecorator(mode, &terminalTextStream{final: "last", err: io.EOF})
			runRehearsalCall(t, llm, rehearsalCall{stream: true}, "first last")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			req := ports.TextRequest{Meta: ports.CallMeta{ForceReplay: mode == "recording"}}
			stream, err := llm.StreamText(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			defer stream.Close()
			cancel()
			if text, err := stream.Recv(); text != "" || !errors.Is(err, context.Canceled) {
				t.Fatalf("replay after cancellation=%q, %v", text, err)
			}
		})
	}
}

func TestStoredDecorators_CanceledRequestsNeverCallProvider(t *testing.T) {
	for _, mode := range []string{"cache", "recording"} {
		t.Run(mode, func(t *testing.T) {
			called := false
			live := fakeLLM{
				jsonFn:   func(context.Context) (json.RawMessage, error) { called = true; return nil, nil },
				streamFn: func(context.Context) (ports.TextStream, error) { called = true; return nil, nil },
			}
			var llm ports.LLM = Cached(live, &memoryCache{values: map[string][]byte{}}, "test")
			if mode == "recording" {
				llm = RecordReplay(live, &memoryRecordings{values: map[ports.RecKey]domain.Recording{}}, "test")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_, jsonErr := llm.JSON(ctx, ports.TextRequest{}, ports.Schema{})
			_, textErr := llm.StreamText(ctx, ports.TextRequest{})
			if called || !errors.Is(jsonErr, context.Canceled) || !errors.Is(textErr, context.Canceled) {
				t.Fatalf("called=%v JSON=%v text=%v", called, jsonErr, textErr)
			}
		})
	}
}
