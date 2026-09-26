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
	reader, err := e.openLocalized(lineCtx, effect.AssetID)
	if err != nil {
		if !isCanceled(lineCtx, err) {
			postLineFailed(ctx, scope, in, effect.UtteranceID, vocab.ErrUnavailable)
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
			}
			previous = append(previous[:0], buf[:n]...)
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				if len(previous) > 0 {
					samples += e.assetFrame(ctx, scope, in, effect.UtteranceID, previous, seq, true, &first)
				}
				postLineFinal(ctx, scope, in, effect.UtteranceID, samples, defaultSampleRate)
				postLineDone(ctx, scope, in, effect.UtteranceID)
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

func (e *CannedExecutor) untrack(id domain.UtteranceID, line *lineControl) {
	e.mu.Lock()
	if current := e.stop[id]; current == line {
		delete(e.stop, id)
	}
	e.mu.Unlock()
}
