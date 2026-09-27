package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type speechProvider struct {
	calls  int
	stream ports.PCMStream
	err    error
}

func (p *speechProvider) Stream(context.Context, ports.TTSRequest, ports.TextStream) (ports.PCMStream, error) {
	p.calls++
	return p.stream, p.err
}

type speechChunks struct {
	chunks   []ports.PCMChunk
	terminal error
	closes   int
}

func (s *speechChunks) Recv() (ports.PCMChunk, error) {
	if len(s.chunks) == 0 {
		return ports.PCMChunk{}, io.EOF
	}
	chunk := s.chunks[0]
	s.chunks = s.chunks[1:]
	if len(s.chunks) == 0 {
		return chunk, s.terminal
	}
	return chunk, nil
}
func (s *speechChunks) Close() error { s.closes++; return nil }

func speechRequest() ports.TTSRequest {
	return ports.TTSRequest{Meta: ports.CallMeta{Role: vocab.RoleNPCReply, Phase: vocab.StateConversation, Seat: 1, Index: 2, Locale: "en"}, VoiceID: "vell", SampleRate: 24000}
}
func speechStore() *memoryRecordings {
	return &memoryRecordings{values: make(map[ports.RecKey]domain.Recording)}
}
func collectSpeech(t *testing.T, stream ports.PCMStream) []ports.PCMChunk {
	t.Helper()
	defer stream.Close()
	var out []ports.PCMChunk
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return out
		}
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, chunk)
	}
}

func TestRecordedTTS_ReplaysTerminalPCMAndSeparatesContracts(t *testing.T) {
	store := speechStore()
	req := speechRequest()
	live := &speechProvider{stream: &speechChunks{chunks: []ports.PCMChunk{{SampleRate: 24000, S16LE: []byte{1, 2}}, {S16LE: []byte{3, 4}}}, terminal: io.EOF}}
	tts := RecordReplayTTS(live, store, "eleven")
	stream, err := tts.Stream(t.Context(), req, newFakeStream("hello"))
	if err != nil {
		t.Fatal(err)
	}
	want := collectSpeech(t, stream)
	if len(want) != 2 || want[1].SampleRate != 24000 || !reflect.DeepEqual(want[1].S16LE, []byte{3, 4}) {
		t.Fatalf("terminal audio=%+v", want)
	}
	req.Meta.ForceReplay = true
	req.Meta.Run = "new-run"
	req.Meta.UtteranceID = "new-line"
	stream, err = tts.Stream(t.Context(), req, newFakeStream("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if got := collectSpeech(t, stream); !reflect.DeepEqual(got, want) || live.calls != 1 {
		t.Fatalf("replay=%+v calls=%d", got, live.calls)
	}
	for _, tc := range []struct {
		name   string
		change func(*ports.TTSRequest)
	}{
		{"voice", func(r *ports.TTSRequest) { r.VoiceID = "stranger" }},
		{"rate", func(r *ports.TTSRequest) { r.SampleRate = 48000 }},
		{"locale", func(r *ports.TTSRequest) { r.Meta.Locale = "es" }},
		{"role", func(r *ports.TTSRequest) { r.Meta.Role = vocab.RoleOpening }},
		{"seat", func(r *ports.TTSRequest) { r.Meta.Seat = 2 }},
		{"phase", func(r *ports.TTSRequest) { r.Meta.Phase = vocab.StateOpening }},
		{"position", func(r *ports.TTSRequest) { r.Meta.Index++ }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			different := req
			tc.change(&different)
			if _, err := tts.Stream(t.Context(), different, newFakeStream("text")); err == nil {
				t.Fatal("wrong speech recording replayed")
			}
		})
	}
	if live.calls != 1 {
		t.Fatal("forced replay miss contacted provider")
	}
}

func TestRecordedTTS_PreRenderedLinesHaveDistinctIdentity(t *testing.T) {
	a := speechRequest()
	a.Meta.Role = ""
	a.Meta.UtteranceID = "prerender:first"
	b := a
	b.Meta.UtteranceID = "prerender:second"
	if ttsRecordingKey("tts", a) == ttsRecordingKey("tts", b) {
		t.Fatal("pre-rendered lines collide")
	}
}

func TestRecordedTTS_FailureAndCancellationKeepPreviousRecording(t *testing.T) {
	for _, name := range []string{"cancel", "close", "failure", "oversized", "invalid", "empty"} {
		t.Run(name, func(t *testing.T) {
			store := speechStore()
			req := speechRequest()
			key := ttsRecordingKey("tts", req)
			old := domain.Recording{Audio: []byte("previous")}
			store.values[key] = old
			source := &speechChunks{chunks: []ports.PCMChunk{{SampleRate: 24000, S16LE: []byte{1, 2}}}, terminal: io.EOF}
			if name == "failure" {
				source.terminal = errors.New("disconnected")
			}
			if name == "oversized" {
				source.chunks[0].S16LE = make([]byte, maxRecordedPCMBytes+2)
			}
			if name == "invalid" {
				source.chunks[0].S16LE = []byte{1}
			}
			if name == "empty" {
				source.chunks = nil
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			stream, err := RecordReplayTTS(&speechProvider{stream: source}, store, "tts").Stream(ctx, req, newFakeStream("text"))
			if err != nil {
				t.Fatal(err)
			}
			if name == "cancel" {
				cancel()
			}
			if name == "close" {
				_ = stream.Close()
			}
			_, err = stream.Recv()
			if (name == "cancel" || name == "failure" || name == "invalid") && err == nil {
				t.Fatal("bad stream succeeded")
			}
			_ = stream.Close()
			_ = stream.Close()
			if source.closes != 1 {
				t.Fatalf("closed source %d times", source.closes)
			}
			if !reflect.DeepEqual(store.values[key], old) {
				t.Fatal("incomplete recording replaced valid audio")
			}
		})
	}
}

func TestRecordedTTS_ReplayMissingCorruptOrCanceledNeverCallsProvider(t *testing.T) {
	for _, tc := range []struct {
		name   string
		record domain.Recording
	}{
		{"missing", domain.Recording{}},
		{"wrong format", domain.Recording{MIME: "audio/mp3", Audio: []byte("data")}},
		{"invalid json", domain.Recording{MIME: pcmRecordingMIME, Audio: []byte("{")}},
		{"empty", domain.Recording{MIME: pcmRecordingMIME, Audio: []byte("[]")}},
		{"odd bytes", domain.Recording{MIME: pcmRecordingMIME, Audio: []byte(`[{"SampleRate":24000,"S16LE":"AQ=="}]`)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := speechStore()
			req := speechRequest()
			req.Meta.ForceReplay = true
			store.values[ttsRecordingKey("tts", req)] = tc.record
			live := &speechProvider{}
			if _, err := RecordReplayTTS(live, store, "tts").Stream(t.Context(), req, newFakeStream("text")); err == nil {
				t.Fatal("invalid replay succeeded")
			}
			if live.calls != 0 {
				t.Fatal("invalid replay contacted provider")
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	live := &speechProvider{}
	if _, err := RecordReplayTTS(live, nil, "tts").Stream(ctx, speechRequest(), nil); !errors.Is(err, context.Canceled) || live.calls != 0 {
		t.Fatalf("canceled=%v calls=%d", err, live.calls)
	}
	req := speechRequest()
	req.Meta.ForceReplay = true
	if _, err := RecordReplayTTS(live, nil, "tts").Stream(t.Context(), req, nil); err == nil || live.calls != 0 {
		t.Fatal("missing store allowed live call")
	}
}

func TestRecordedTTS_UsesRecordingOnInitialProviderFailure(t *testing.T) {
	store := speechStore()
	req := speechRequest()
	data, _ := json.Marshal([]ports.PCMChunk{{SampleRate: 24000, S16LE: []byte{9, 8}}})
	store.values[ttsRecordingKey("tts", req)] = domain.Recording{MIME: pcmRecordingMIME, Audio: data}
	for _, next := range []ports.TTS{nil, &speechProvider{err: errors.New("offline")}, &speechProvider{}} {
		stream, err := RecordReplayTTS(next, store, "tts").Stream(t.Context(), req, nil)
		if err != nil {
			t.Fatal(err)
		}
		if got := collectSpeech(t, stream); len(got) != 1 || got[0].S16LE[0] != 9 {
			t.Fatalf("fallback=%+v", got)
		}
	}
}

type cancelPCM struct{ unblock chan struct{} }

func (s *cancelPCM) Recv() (ports.PCMChunk, error) {
	<-s.unblock
	return ports.PCMChunk{SampleRate: 24000, S16LE: []byte{1, 2}}, io.EOF
}
func (s *cancelPCM) Close() error { close(s.unblock); return nil }

func TestRecordedPCM_CloseInterruptsPendingReadWithoutSaving(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		saved := false
		source := &cancelPCM{unblock: make(chan struct{})}
		stream := newRecordedPCM(t.Context(), source, 24000, func(context.Context, []ports.PCMChunk) error { saved = true; return nil })
		result := make(chan error, 1)
		go func() { _, err := stream.Recv(); result <- err }()
		synctest.Wait()
		_ = stream.Close()
		if err := <-result; !errors.Is(err, context.Canceled) || saved {
			t.Fatalf("close=%v saved=%v", err, saved)
		}
	})
}
