package audio

import (
	"errors"
	"time"

	dungeonfluxv1 "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

const (
	// JitterLead is the amount of PCM queued before the first buffer starts.
	JitterLead     = 150 * time.Millisecond
	bytesPerSample = 2
)

// ScheduledChunk is one mono PCM frame with its playback start offset.
type ScheduledChunk struct {
	UtteranceID string
	PCM         []byte
	SampleRate  int
	Start       time.Duration
	Final       bool
}

// Scheduler converts protocol audio messages into contiguous playback plans.
// The zero value is ready for use.
type Scheduler struct {
	lines map[string]time.Duration
}

// Accept consumes a frame and returns its scheduled chunk. Invalid frames and
// frames cancelled earlier are discarded.
func (s *Scheduler) Accept(frame *dungeonfluxv1.AudioFrame) (ScheduledChunk, bool, error) {
	if frame == nil || frame.GetUtteranceId() == "" {
		return ScheduledChunk{}, false, errors.New("audio: frame has no utterance id")
	}
	if frame.GetSampleRate() <= 0 || len(frame.GetPcmS16Le())%bytesPerSample != 0 {
		return ScheduledChunk{}, false, errors.New("audio: invalid PCM frame")
	}
	if s.lines == nil {
		s.lines = make(map[string]time.Duration)
	}
	start, ok := s.lines[frame.GetUtteranceId()]
	if !ok {
		start = JitterLead
	}
	chunk := ScheduledChunk{
		UtteranceID: frame.GetUtteranceId(),
		PCM:         append([]byte(nil), frame.GetPcmS16Le()...),
		SampleRate:  int(frame.GetSampleRate()),
		Start:       start,
		Final:       frame.GetFinal(),
	}
	duration := time.Duration(len(frame.GetPcmS16Le())/bytesPerSample) * time.Second / time.Duration(frame.GetSampleRate())
	if frame.GetFinal() {
		delete(s.lines, frame.GetUtteranceId())
	} else {
		s.lines[frame.GetUtteranceId()] = start + duration
	}
	return chunk, true, nil
}

// Cancel drops all queued timing for an utterance. An empty ID cancels all
// utterances, matching AudioCancel{all}.
func (s *Scheduler) Cancel(utteranceID string) {
	if utteranceID == "" {
		s.lines = nil
		return
	}
	delete(s.lines, utteranceID)
}

// Pending reports whether the scheduler has a non-final line queued.
func (s *Scheduler) Pending(utteranceID string) bool {
	_, ok := s.lines[utteranceID]
	return ok
}
