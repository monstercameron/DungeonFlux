package fakes

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/clock"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// LLMCall records one request made to FakeLLM.
type LLMCall struct {
	Request ports.TextRequest
	Schema  ports.Schema
	JSON    bool
}

// LLMTextResult scripts one StreamText response.
type LLMTextResult struct {
	Chunks []string
	Err    error
	Delay  time.Duration
}

// LLMJSONResult scripts one JSON response.
type LLMJSONResult struct {
	Value json.RawMessage
	Err   error
	Delay time.Duration
}

// FakeLLM is a scripted LLM fake.
type FakeLLM struct {
	Clock      clock.Clock
	Text       []LLMTextResult
	JSONScript []LLMJSONResult
	TextCalls  []LLMCall
	JSONCalls  []LLMCall
	mu         sync.Mutex
}

// StreamText returns the next scripted text stream.
func (f *FakeLLM) StreamText(ctx context.Context, req ports.TextRequest) (ports.TextStream, error) {
	f.mu.Lock()
	f.TextCalls = append(f.TextCalls, LLMCall{Request: req})
	r := LLMTextResult{}
	if len(f.Text) > 0 {
		r, f.Text = f.Text[0], f.Text[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return nil, err
	}
	if r.Err != nil {
		return nil, r.Err
	}
	return &textStream{chunks: append([]string(nil), r.Chunks...), err: io.EOF}, nil
}

// JSON returns the next scripted JSON response.
func (f *FakeLLM) JSON(ctx context.Context, req ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	f.mu.Lock()
	f.JSONCalls = append(f.JSONCalls, LLMCall{Request: req, Schema: schema, JSON: true})
	r := LLMJSONResult{}
	if len(f.JSONScript) > 0 {
		r, f.JSONScript = f.JSONScript[0], f.JSONScript[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return nil, err
	}
	return append(json.RawMessage(nil), r.Value...), r.Err
}

// ImageResult scripts one image stream response.
type ImageResult struct {
	Events []ports.ImageEvent
	Err    error
	Delay  time.Duration
}

// ImageCall records one image request.
type ImageCall struct{ Request ports.ImageRequest }

// FakeImageGen is a scripted image generator fake.
type FakeImageGen struct {
	Clock  clock.Clock
	Script []ImageResult
	Calls  []ImageCall
	mu     sync.Mutex
}

// Generate returns the next scripted image stream.
func (f *FakeImageGen) Generate(ctx context.Context, req ports.ImageRequest) (ports.ImageStream, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, ImageCall{req})
	r := ImageResult{}
	if len(f.Script) > 0 {
		r, f.Script = f.Script[0], f.Script[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return nil, err
	}
	if r.Err != nil {
		return nil, r.Err
	}
	return &fakeImageStream{events: append([]ports.ImageEvent(nil), r.Events...), err: io.EOF}, nil
}

type fakeImageStream struct {
	mu     sync.Mutex
	events []ports.ImageEvent
	index  int
	err    error
	closed bool
}

func (s *fakeImageStream) Recv() (ports.ImageEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ports.ImageEvent{}, io.ErrClosedPipe
	}
	if s.index == len(s.events) {
		return ports.ImageEvent{}, s.err
	}
	v := s.events[s.index]
	s.index++
	return v, nil
}
func (s *fakeImageStream) Close() error { s.mu.Lock(); s.closed = true; s.mu.Unlock(); return nil }

// VideoResult scripts one Submit or Poll response.
type VideoResult struct {
	Job    ports.VideoJob
	Status ports.VideoStatus
	Err    error
	Delay  time.Duration
}

// VideoCall records video requests and jobs.
type VideoCall struct {
	Request *ports.VideoRequest
	Job     ports.VideoJob
}

// FakeVideoGen is a scripted video generator fake.
type FakeVideoGen struct {
	Clock   clock.Clock
	Submits []VideoResult
	Polls   []VideoResult
	Calls   []VideoCall
	mu      sync.Mutex
}

// Submit returns the next scripted job.
func (f *FakeVideoGen) Submit(ctx context.Context, req ports.VideoRequest) (ports.VideoJob, error) {
	f.mu.Lock()
	r := VideoResult{}
	if len(f.Submits) > 0 {
		r, f.Submits = f.Submits[0], f.Submits[1:]
	}
	x := req
	f.Calls = append(f.Calls, VideoCall{Request: &x})
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return ports.VideoJob{}, err
	}
	return r.Job, r.Err
}

// Poll returns the next scripted status.
func (f *FakeVideoGen) Poll(ctx context.Context, job ports.VideoJob) (ports.VideoStatus, error) {
	f.mu.Lock()
	r := VideoResult{}
	if len(f.Polls) > 0 {
		r, f.Polls = f.Polls[0], f.Polls[1:]
	}
	f.Calls = append(f.Calls, VideoCall{Job: job})
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return ports.VideoStatus{}, err
	}
	return r.Status, r.Err
}

// TTSResult scripts one TTS response.
type TTSResult struct {
	Chunks []ports.PCMChunk
	Err    error
	Delay  time.Duration
}

// TTSCall records one TTS request.
type TTSCall struct {
	Request ports.TTSRequest
	Text    ports.TextStream
}

// FakeTTS is a scripted text-to-speech fake.
type FakeTTS struct {
	Clock  clock.Clock
	Script []TTSResult
	Calls  []TTSCall
	mu     sync.Mutex
}

// Stream returns the next scripted PCM stream.
func (f *FakeTTS) Stream(ctx context.Context, req ports.TTSRequest, text ports.TextStream) (ports.PCMStream, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, TTSCall{req, text})
	r := TTSResult{}
	if len(f.Script) > 0 {
		r, f.Script = f.Script[0], f.Script[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return nil, err
	}
	if r.Err != nil {
		return nil, r.Err
	}
	return &pcmStream{chunks: append([]ports.PCMChunk(nil), r.Chunks...), err: io.EOF}, nil
}

type pcmStream struct {
	mu     sync.Mutex
	chunks []ports.PCMChunk
	index  int
	err    error
	closed bool
}

func (s *pcmStream) Recv() (ports.PCMChunk, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ports.PCMChunk{}, io.ErrClosedPipe
	}
	if s.index == len(s.chunks) {
		return ports.PCMChunk{}, s.err
	}
	v := s.chunks[s.index]
	s.index++
	return v, nil
}
func (s *pcmStream) Close() error { s.mu.Lock(); s.closed = true; s.mu.Unlock(); return nil }

// STTResult scripts one transcription.
type STTResult struct {
	Transcript ports.Transcript
	Err        error
	Delay      time.Duration
}

// FakeSTT is a scripted speech-to-text fake.
type FakeSTT struct {
	Clock  clock.Clock
	Script []STTResult
	Calls  []ports.STTRequest
	mu     sync.Mutex
}

// Transcribe returns the next scripted transcript.
func (f *FakeSTT) Transcribe(ctx context.Context, req ports.STTRequest) (ports.Transcript, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, req)
	r := STTResult{}
	if len(f.Script) > 0 {
		r, f.Script = f.Script[0], f.Script[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return ports.Transcript{}, err
	}
	return r.Transcript, r.Err
}

// SoundResult scripts one generated sound.
type SoundResult struct {
	Sound ports.Sound
	Err   error
	Delay time.Duration
}

// FakeSoundGen is a scripted sound generator fake.
type FakeSoundGen struct {
	Clock  clock.Clock
	Script []SoundResult
	Calls  []ports.SoundRequest
	mu     sync.Mutex
}

// Generate returns the next scripted sound.
func (f *FakeSoundGen) Generate(ctx context.Context, req ports.SoundRequest) (ports.Sound, error) {
	f.mu.Lock()
	f.Calls = append(f.Calls, req)
	r := SoundResult{}
	if len(f.Script) > 0 {
		r, f.Script = f.Script[0], f.Script[1:]
	}
	f.mu.Unlock()
	if err := wait(ctx, f.Clock, r.Delay); err != nil {
		return ports.Sound{}, err
	}
	return r.Sound, r.Err
}
