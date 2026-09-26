package phone

import (
	"context"
	"errors"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
	"google.golang.org/grpc"
)

const audioQueueCapacity = 32

// PhoneCue identifies the short effects used by the player phone.
type PhoneCue string

const (
	// CueRoll is played when the player's check is rolled.
	CueRoll PhoneCue = "roll"
	// CueTurn is played when it becomes the player's turn.
	CueTurn PhoneCue = "your_turn"
	// CueDamage is played when the player's character takes damage.
	CueDamage PhoneCue = "damage"
	// CueDown is played when the player's character drops.
	CueDown PhoneCue = "down"
	// CueSuccess is played after a successful check or attack.
	CueSuccess PhoneCue = "success"
	// CueFailure is played after a failed check or attack.
	CueFailure PhoneCue = "failure"
)

// PhoneAudioMessage is a queued, seat-relevant one-off audio message.
type PhoneAudioMessage struct {
	Message *df.AudioMessage
	Cue     PhoneCue
}

// AudioQueue is a bounded queue for short phone effects. It is safe for the
// stream receiver and browser playback loop to use from separate goroutines.
type AudioQueue struct {
	mu       sync.Mutex
	items    []PhoneAudioMessage
	muted    bool
	maxItems int
}

// NewAudioQueue creates a queue with the phone's fixed back-pressure limit.
func NewAudioQueue() *AudioQueue {
	return &AudioQueue{maxItems: audioQueueCapacity}
}

// SetMuted changes whether newly received cues are discarded.
func (q *AudioQueue) SetMuted(muted bool) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.muted = muted
}

// Muted reports whether the queue is suppressing cues.
func (q *AudioQueue) Muted() bool {
	if q == nil {
		return true
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.muted
}

// Push queues a matching SFX message without blocking the stream receiver.
func (q *AudioQueue) Push(message *df.AudioMessage, seat int32) bool {
	if q == nil || !isPhoneSFX(message, seat) {
		return false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.muted || len(q.items) >= q.maxItems {
		return false
	}
	q.items = append(q.items, PhoneAudioMessage{Message: message, Cue: cueFor(message)})
	return true
}

// Pop removes the oldest queued cue, returning false when the queue is empty.
func (q *AudioQueue) Pop() (PhoneAudioMessage, bool) {
	if q == nil {
		return PhoneAudioMessage{}, false
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return PhoneAudioMessage{}, false
	}
	item := q.items[0]
	q.items[0] = PhoneAudioMessage{}
	q.items = q.items[1:]
	return item, true
}

// Len reports the number of pending cues.
func (q *AudioQueue) Len() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

// PhoneAudio owns phone cue policy and is deliberately independent of the
// browser so its filtering and mute behaviour can be tested natively.
type PhoneAudio struct {
	queue          *AudioQueue
	reducedMotion  bool
	vibrateEnabled bool
	service        phoneAudioService
	seatToken      string
	seat           int32
	ctx            context.Context
	cancel         context.CancelFunc
}

type phoneAudioService interface {
	Listen(context.Context, *df.ListenRequest, ...grpc.CallOption) (grpc.ServerStreamingClient[df.AudioMessage], error)
}

// NewPhoneAudio creates phone audio policy with haptics enabled.
func NewPhoneAudio() *PhoneAudio {
	return &PhoneAudio{queue: NewAudioQueue(), vibrateEnabled: true}
}

// SetMuted suppresses future cues until unmuted.
func (a *PhoneAudio) SetMuted(muted bool) {
	if a != nil && a.queue != nil {
		a.queue.SetMuted(muted)
	}
}

// SetReducedMotion disables haptics while retaining audio playback.
func (a *PhoneAudio) SetReducedMotion(reduced bool) {
	if a != nil {
		a.reducedMotion = reduced
		a.vibrateEnabled = !reduced
	}
}

// HapticsEnabled reports whether a cue may request a vibration.
func (a *PhoneAudio) HapticsEnabled() bool {
	return a != nil && a.vibrateEnabled && !a.reducedMotion
}

// Enqueue applies target, channel, mute, and queue policies.
func (a *PhoneAudio) Enqueue(message *df.AudioMessage, seat int32) bool {
	if a == nil {
		return false
	}
	return a.queue.Push(message, seat)
}

// Next returns the next cue for browser playback.
func (a *PhoneAudio) Next() (PhoneAudioMessage, bool) {
	if a == nil {
		return PhoneAudioMessage{}, false
	}
	return a.queue.Pop()
}

func isPhoneSFX(message *df.AudioMessage, seat int32) bool {
	if message == nil || message.GetChannel() != df.AudioChannel_AUDIO_CHANNEL_SFX || message.GetChunk() == nil {
		return false
	}
	target := message.GetTarget()
	if target == nil {
		return false
	}
	switch target.GetKind() {
	case df.AudioTargetKind_AUDIO_TARGET_KIND_ALL_PHONES:
		return true
	case df.AudioTargetKind_AUDIO_TARGET_KIND_SEAT:
		return seat > 0 && target.GetSeat() == seat
	default:
		return false
	}
}

func cueFor(message *df.AudioMessage) PhoneCue {
	if message == nil || message.GetChunk() == nil {
		return ""
	}
	return PhoneCue(message.GetChunk().GetCodecMime())
}

var errAudioMessage = errors.New("phone audio: unsupported message")

func validateAudioMessage(message *df.AudioMessage) error {
	if !isPhoneSFX(message, 1) && !isPhoneSFX(message, 2) {
		return errAudioMessage
	}
	return nil
}
