package out

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/i18n"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

const cannedChunkSize = 4096

// AssetReader supplies the bytes of a stored audio asset. It is deliberately
// local until the shared storage contract grows a byte-stream operation.
type AssetReader interface {
	Open(context.Context, domain.AssetID) (io.ReadCloser, error)
}

// CannedExecutor streams pre-rendered PCM through the same Listen port as live
// lines.
type CannedExecutor struct {
	assets AssetReader
	audio  ports.AudioOut
	mu     sync.Mutex
	stop   map[domain.UtteranceID]*lineControl
	// Room reports the room-default locale for per-locale canned audio. A
	// nil source keeps English. Composition wires it to room state.
	Room ports.RoomLocale
}

// NewCannedExecutor creates a canned-line executor.
func NewCannedExecutor(assets AssetReader, audio ports.AudioOut) *CannedExecutor {
	return &CannedExecutor{assets: assets, audio: audio, stop: make(map[domain.UtteranceID]*lineControl)}
}

// PlayCanned streams the selected asset and posts line_done after its final
// PCM frame.
func (e *CannedExecutor) PlayCanned(ctx context.Context, effect domain.PlayCanned, scope domain.Scope, in ports.Inbox) {
	if ctx == nil {
		ctx = context.Background()
	}
	if e == nil || e.assets == nil || e.audio == nil {
		postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
		return
	}
	lineCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	line := e.track(effect.UtteranceID, cancel)
	defer e.untrack(effect.UtteranceID, line)
	completed := false
	defer func() {
		if !completed && lineCtx.Err() != nil {
			e.interrupt(effect.UtteranceID, line)
		}
	}()
	text := cannedText(e.roomLocale(), effect.AssetID)
	postNarrationText(ctx, scope, in, effect.UtteranceID, cannedSpeaker(effect), text, false)
	reader, err := e.openLocalized(lineCtx, effect.AssetID)
	if err != nil {
		if !isCanceled(lineCtx, err) {
			// Build-time media is optional in fake mode and may be absent during
			// early rehearsals. A short silence preserves the line lifecycle so
			// the engine can continue instead of waiting forever.
			e.playSilence(ctx, scope, in, effect)
		}
		return
	}
	defer reader.Close()
	go closeOnCancelReader(lineCtx, reader)
	buf := make([]byte, cannedChunkSize)
	var previous []byte
	seq, samples := 0, 0
	first := true
	for {
		n, readErr := reader.Read(buf)
		if n > 0 {
			if len(previous) > 0 {
				samples += e.assetFrame(ctx, scope, in, effect.UtteranceID, previous, seq, false, &first)
				seq++
				// Pace frames in real time: emitting a whole recording at once
				// overran the Listen hub's bounded buffer and dropped the TV.
				if !sleepCtx(lineCtx, time.Duration(len(previous))*time.Second/(defaultSampleRate*2)) {
					return
				}
			}
			previous = append(previous[:0], buf[:n]...)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if len(previous) > 0 {
					samples += e.assetFrame(ctx, scope, in, effect.UtteranceID, previous, seq, true, &first)
				}
				postNarrationText(ctx, scope, in, effect.UtteranceID, cannedSpeaker(effect), text, true)
				postLineFinal(ctx, scope, in, effect.UtteranceID, samples, defaultSampleRate)
				postLineDone(ctx, scope, in, effect.UtteranceID)
				completed = true
			}
			if !errors.Is(readErr, io.EOF) && !isCanceled(lineCtx, readErr) {
				postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
			}
			return
		}
		select {
		case <-lineCtx.Done():
			return
		default:
		}
	}
}

func (e *CannedExecutor) playSilence(ctx context.Context, scope domain.Scope, in ports.Inbox, effect domain.PlayCanned) {
	timer := time.NewTimer(1200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ctx.Done():
		return
	}
	text := cannedText(e.roomLocale(), effect.AssetID)
	e.audio.Frame(domain.AudioFrame{UtteranceID: effect.UtteranceID, SampleRate: defaultSampleRate, PCMS16LE: make([]byte, defaultSampleRate*2*5/4), Final: true})
	postNarrationText(ctx, scope, in, effect.UtteranceID, cannedSpeaker(effect), text, true)
	postLineFirst(ctx, scope, in, effect.UtteranceID)
	postLineFinal(ctx, scope, in, effect.UtteranceID, defaultSampleRate*5/4, defaultSampleRate)
	postLineDone(ctx, scope, in, effect.UtteranceID)
}

func cannedText(locale string, id domain.AssetID) string {
	value := i18n.Default().T(locale, "canned."+string(id), nil, 0, "")
	if value == "canned."+string(id) {
		return ""
	}
	return strings.TrimSpace(value)
}

func cannedSpeaker(effect domain.PlayCanned) string {
	if effect.Speaker != "" {
		return effect.Speaker
	}
	switch {
	case strings.Contains(string(effect.AssetID), "npc") || strings.Contains(string(effect.AssetID), "nudge_conversation"):
		return "Mother Vell"
	case strings.Contains(string(effect.AssetID), "stranger"):
		return "Stranger"
	default:
		return "Dungeon Master"
	}
}

func (e *CannedExecutor) assetFrame(ctx context.Context, scope domain.Scope, in ports.Inbox, id domain.UtteranceID, data []byte, seq int, final bool, first *bool) int {
	e.audio.Frame(domain.AudioFrame{UtteranceID: id, Seq: seq, SampleRate: defaultSampleRate, PCMS16LE: append([]byte(nil), data...), Final: final})
	if *first {
		postLineFirst(ctx, scope, in, id)
		*first = false
	}
	return len(data) / 2
}

func closeOnCancelReader(ctx context.Context, reader io.ReadCloser) {
	<-ctx.Done()
	_ = reader.Close()
}

// Execute is an alias suitable for executor registration by composition code.
func (e *CannedExecutor) Execute(ctx context.Context, effect domain.PlayCanned, scope domain.Scope, in ports.Inbox) {
	e.PlayCanned(ctx, effect, scope, in)
}

// Cancel stops the active canned line and removes queued Listen audio.
func (e *CannedExecutor) Cancel(id domain.UtteranceID) {
	if e == nil {
		return
	}
	e.mu.Lock()
	line := e.stop[id]
	delete(e.stop, id)
	e.mu.Unlock()
	if line != nil {
		line.cancel()
	}
	if e.audio != nil {
		e.audio.Cancel(id)
	}
}

func (e *CannedExecutor) track(id domain.UtteranceID, cancel context.CancelFunc) *lineControl {
	e.mu.Lock()
	if previous := e.stop[id]; previous != nil {
		previous.cancel()
	}
	line := &lineControl{cancel: cancel}
	e.stop[id] = line
	e.mu.Unlock()
	return line
}

// interrupt fades out a canned line stopped before its end; see
// PCMExecutor.interrupt.
func (e *CannedExecutor) interrupt(id domain.UtteranceID, line *lineControl) {
	e.mu.Lock()
	current := e.stop[id] == line
	e.mu.Unlock()
	if current && e.audio != nil {
		e.audio.Cancel(id)
	}
}

func (e *CannedExecutor) untrack(id domain.UtteranceID, line *lineControl) {
	e.mu.Lock()
	if current := e.stop[id]; current == line {
		delete(e.stop, id)
	}
	e.mu.Unlock()
}

// sleepCtx waits d or until ctx ends; it reports whether the wait completed.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
