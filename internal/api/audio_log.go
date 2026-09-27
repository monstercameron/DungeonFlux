package api

import (
	"log/slog"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

// talkChunkMS is the phone recorder's timeslice: one AudioChunk per 100 ms.
const talkChunkMS = 100

// talkStats counts one recording's audio as it arrives.
type talkStats struct {
	chunks int
	bytes  int
}

func (t *talkStats) add(size int) {
	t.chunks++
	t.bytes += size
}

// SetLogger enables talk logging: one line when a recording starts and one
// when it ends, is cancelled, or fails, with its chunk and byte counts.
func (s *TalkServer) SetLogger(logger *slog.Logger) {
	if s != nil && logger != nil {
		s.logger = logger.With("component", "talk")
	}
}

func (s *TalkServer) log() *slog.Logger {
	if s == nil || s.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return s.logger
}

func (s *TalkServer) logTalk(msg string, session TalkSession, stats talkStats, extra ...any) {
	args := []any{"seat", int(session.Seat), "utterance", string(session.UtteranceID), "chunks", stats.chunks, "bytes", stats.bytes, "audio_ms", stats.chunks * talkChunkMS}
	s.log().Info(msg, append(args, extra...)...)
}

// lineStats counts one voice line's frames as the hub fans them out.
type lineStats struct {
	frames    int
	bytes     int
	listeners int
}

// SetLogger enables listen logging: subscriptions opening and closing, a
// summary per voice line (frames, bytes, listeners reached), and listeners
// dropped for lagging.
func (h *ListenHub) SetLogger(logger *slog.Logger) {
	if h == nil || logger == nil {
		return
	}
	h.mu.Lock()
	h.logger = logger.With("component", "listen")
	h.mu.Unlock()
}

func (h *ListenHub) log() *slog.Logger {
	if h == nil || h.logger == nil {
		return slog.New(slog.DiscardHandler)
	}
	return h.logger
}

// countFrame records a voice frame and returns the finished line's stats when
// the frame is the line's last. Callers hold h.mu.
func (h *ListenHub) countFrame(frame domain.AudioFrame, listeners int) (lineStats, bool) {
	if h.lines == nil {
		h.lines = make(map[domain.UtteranceID]lineStats)
	}
	line := h.lines[frame.UtteranceID]
	line.frames++
	line.bytes += len(frame.PCMS16LE)
	if listeners > line.listeners {
		line.listeners = listeners
	}
	if !frame.Final {
		h.lines[frame.UtteranceID] = line
		return lineStats{}, false
	}
	delete(h.lines, frame.UtteranceID)
	return line, true
}

func (h *ListenHub) voiceListeners() int {
	count := 0
	for _, sub := range h.subscribers {
		if !sub.closed() && sub.accepts(AudioMessage{Channel: AudioVoice, Target: AudioTarget{Kind: TargetDM}}) {
			count++
		}
	}
	return count
}
