// Package out executes voice output effects and publishes PCM to the Listen
// audio port.
package out

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const defaultSampleRate = 24000

// PCMExecutor turns StartLine effects into Listen PCM frames.
type PCMExecutor struct {
	tts   ports.TTS
	audio ports.AudioOut
	mu    sync.Mutex
	stop  map[domain.UtteranceID]*lineControl
	// Room reports the room-default locale for voice selection. A nil
	// source keeps English. Composition wires it to room state.
	Room ports.RoomLocale
	// Voices overrides the TTS voice per locale tag. When a locale has no
	// entry, the effect voice is used, or the locale default voice.
	Voices map[string]string
}

// NewPCMExecutor creates a line executor using tts and audio as its output
// ports.
func NewPCMExecutor(tts ports.TTS, audio ports.AudioOut) *PCMExecutor {
	return &PCMExecutor{tts: tts, audio: audio, stop: make(map[domain.UtteranceID]*lineControl)}
}

// StartLine executes one streaming TTS line and posts its lifecycle events.
func (e *PCMExecutor) StartLine(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil || e.tts == nil || e.audio == nil {
		postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
		return
	}
	lineCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	line := e.track(effect.UtteranceID, cancel)
	defer e.untrack(effect.UtteranceID, line)
	stream, err := e.tts.Stream(lineCtx, ports.TTSRequest{
		Meta:       ports.CallMeta{UtteranceID: effect.UtteranceID, Locale: e.roomLocale()},
		VoiceID:    e.voiceFor(effect.Voice),
		SampleRate: defaultSampleRate,
	}, newTextStream(effect.Input))
	if err != nil {
		if !isCanceled(lineCtx, err) {
			postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
		}
		return
	}
	if stream == nil {
		postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrBadOutput)
		return
	}
	defer stream.Close()
	go closeOnCancel(lineCtx, stream)
	e.consume(lineCtx, stream, effect, scope, in)
}

// Execute is an alias suitable for executor registration by composition code.
func (e *PCMExecutor) Execute(ctx context.Context, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	e.StartLine(ctx, effect, scope, in)
}

// Cancel stops the active line and removes its queued Listen audio.
func (e *PCMExecutor) Cancel(id domain.UtteranceID) {
	if e == nil {
		return
	}
	e.mu.Lock()
	line := e.stop[id]
	e.mu.Unlock()
	if line != nil {
		line.cancel()
	}
	if e.audio != nil {
		e.audio.Cancel(id)
	}
}

func (e *PCMExecutor) track(id domain.UtteranceID, cancel context.CancelFunc) *lineControl {
	e.mu.Lock()
	if previous := e.stop[id]; previous != nil {
		previous.cancel()
	}
	line := &lineControl{cancel: cancel}
	e.stop[id] = line
	e.mu.Unlock()
	return line
}

func (e *PCMExecutor) untrack(id domain.UtteranceID, line *lineControl) {
	e.mu.Lock()
	if current := e.stop[id]; current == line {
		delete(e.stop, id)
	}
	e.mu.Unlock()
}

type lineControl struct{ cancel context.CancelFunc }

func (e *PCMExecutor) consume(ctx context.Context, stream ports.PCMStream, effect domain.StartLine, scope domain.Scope, in ports.Inbox) {
	var previous ports.PCMChunk
	seq, samples := 0, 0
	havePrevious := false
	first := true
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			if havePrevious {
				samples += e.frame(ctx, scope, in, effect, previous, seq, true, first)
			}
			postLineFinal(ctx, scope, in, effect.UtteranceID, samples, previous.SampleRate)
			postLineDone(ctx, scope, in, effect.UtteranceID)
			return
		}
		if err != nil {
			if !isCanceled(ctx, err) {
				postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
			}
			return
		}
		if chunk.SampleRate <= 0 || len(chunk.S16LE) == 0 {
			continue
		}
		if havePrevious {
			samples += e.frame(ctx, scope, in, effect, previous, seq, false, first)
			seq++
			first = false
		}
		previous, havePrevious = chunk, true
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (e *PCMExecutor) frame(ctx context.Context, scope domain.Scope, in ports.Inbox, effect domain.StartLine, chunk ports.PCMChunk, seq int, final, first bool) int {
	if first {
		postLineFirst(ctx, scope, in, effect.UtteranceID)
	}
	e.audio.Frame(domain.AudioFrame{UtteranceID: effect.UtteranceID, Speaker: string(effect.Role), Seq: seq, SampleRate: chunk.SampleRate, PCMS16LE: append([]byte(nil), chunk.S16LE...), Final: final})
	return len(chunk.S16LE) / 2
}

func closeOnCancel(ctx context.Context, stream ports.PCMStream) {
	<-ctx.Done()
	_ = stream.Close()
}

func isCanceled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil
}

func postLineFirst(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFirstAudio{UtteranceID: id}})
	}
}

func postLineFinal(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID, samples, rate int) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineAudioFinal{UtteranceID: id, Samples: samples, SampleRate: rate}})
	}
}

func postLineDone(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineDone{UtteranceID: id}})
	}
}

func postLineFailed(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID, kind vocab.ErrKind) {
	if in != nil {
		in.Post(ctx, domain.Envelope{Scope: scope, Event: domain.LineFailed{UtteranceID: id, FailureKind: kind}})
	}
}

type textStream struct {
	text string
	done bool
}

func newTextStream(text string) *textStream { return &textStream{text: text} }

func (s *textStream) Recv() (string, error) {
	if s.done {
		return "", io.EOF
	}
	s.done = true
	return s.text, nil
}

func (s *textStream) Close() error { s.done = true; return nil }
