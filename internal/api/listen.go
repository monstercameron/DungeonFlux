package api

import (
	"context"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

const listenLagLimitMS int64 = 2000

// AudioChannel identifies the mixer bus for a streamed audio message.
type AudioChannel string

const (
	// AudioVoice carries PCM speech and is never dropped in favour of effects.
	AudioVoice AudioChannel = "voice"
	// AudioMusic carries an encoded music chunk or music mix command.
	AudioMusic AudioChannel = "music"
	// AudioAmbience carries an encoded environmental bed.
	AudioAmbience AudioChannel = "ambience"
	// AudioSFX carries a short sound effect.
	AudioSFX AudioChannel = "sfx"
)

// AudioTarget identifies the client(s) that should receive a message.
type AudioTarget struct {
	Kind string
	Seat int
}

const (
	// TargetDM addresses the DM screen.
	TargetDM = "dm"
	// TargetSeat addresses one phone seat.
	TargetSeat = "seat"
	// TargetAllPhones addresses every connected phone.
	TargetAllPhones = "all_phones"
)

// EncodedAudioChunk is one independently transportable encoded audio segment.
type EncodedAudioChunk struct {
	CodecMIME  string
	Seq        uint64
	Data       []byte
	Final      bool
	DurationMS int
}

// MixCommand describes a client-side play, stop, crossfade, or duck action.
type MixCommand struct {
	Command    string
	TrackID    string
	StartAtMS  int64
	DurationMS int
	Loop       bool
	Gain       float32
	Duck       float32
}

// AudioMessage is the hub's transport-neutral audio envelope.
type AudioMessage struct {
	Channel   AudioChannel
	Target    AudioTarget
	Frame     *domain.AudioFrame
	Chunk     *EncodedAudioChunk
	Mix       *MixCommand
	CancelID  string
	CancelAll bool
}

// ListenHub fans PCM frames out to DM listeners. A listener whose unread
// queue exceeds two seconds is removed so a stale tab cannot retain memory.
type ListenHub struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[uint64]*ListenSubscription
	latest      *ListenSubscription
}

// ListenSubscription is one bounded listener queue owned by a client.
type ListenSubscription struct {
	mu       sync.Mutex
	frames   chan domain.AudioFrame
	messages chan AudioMessage
	done     chan struct{}
	dropped  bool
	pending  int64
	remove   func()
	target   AudioTarget
	phone    bool
}

// NewListenHub creates an empty PCM fan-out hub.
func NewListenHub() *ListenHub {
	return &ListenHub{subscribers: make(map[uint64]*ListenSubscription)}
}

// Subscribe adds a listener. Cancellation of ctx removes the listener on its
// next hub operation; no goroutine is created for the subscription.
func (h *ListenHub) Subscribe(ctx context.Context) *ListenSubscription {
	return h.SubscribeTarget(ctx, AudioTarget{Kind: TargetDM}, false)
}

// SubscribeTarget adds a listener filtered to target. Phone listeners receive
// only SFX messages; DM listeners receive all messages addressed to the DM.
func (h *ListenHub) SubscribeTarget(ctx context.Context, target AudioTarget, phone bool) *ListenSubscription {
	if ctx == nil {
		ctx = context.Background()
	}
	h.mu.Lock()
	h.nextID++
	id := h.nextID
	sub := &ListenSubscription{frames: make(chan domain.AudioFrame, 64), messages: make(chan AudioMessage, 64), done: make(chan struct{}), target: target, phone: phone}
	sub.remove = func() { h.remove(id, sub) }
	var previous *ListenSubscription
	for _, current := range h.subscribers {
		if current.target == target && current.phone == phone {
			previous = current
			break
		}
	}
	if !phone {
		h.latest = sub
	}
	h.subscribers[id] = sub
	h.mu.Unlock()
	if previous != nil {
		previous.remove()
	}
	if ctx.Err() != nil {
		sub.Close()
	}
	return sub
}

// Frame publishes one PCM frame, copying its bytes for each subscriber.
func (h *ListenHub) Frame(frame domain.AudioFrame) {
	h.Publish(AudioMessage{Channel: AudioVoice, Target: AudioTarget{Kind: TargetDM}, Frame: &frame})
	duration := frameDurationMS(frame)
	h.mu.Lock()
	deferred := make([]*ListenSubscription, 0)
	for id, sub := range h.subscribers {
		if sub.closed() || sub.queueLegacy(frame, duration) {
			deferred = append(deferred, sub)
			delete(h.subscribers, id)
		}
	}
	h.mu.Unlock()
	for _, sub := range deferred {
		sub.finish()
	}
}

// Publish sends an encoded chunk, mix command, or targeted voice message to
// matching listeners. A full non-voice queue drops its oldest non-voice item.
func (h *ListenHub) Publish(message AudioMessage) {
	if message.Channel == "" {
		message.Channel = AudioVoice
	}
	h.mu.Lock()
	deferred := make([]*ListenSubscription, 0)
	for id, sub := range h.subscribers {
		if sub.closed() || !sub.accepts(message) {
			continue
		}
		if sub.queueMessage(message) {
			deferred = append(deferred, sub)
			delete(h.subscribers, id)
		}
	}
	h.mu.Unlock()
	for _, sub := range deferred {
		sub.finish()
	}
}

// Cancel removes queued frames for one utterance from every listener.
func (h *ListenHub) Cancel(utteranceID domain.UtteranceID) {
	h.mu.Lock()
	for _, sub := range h.subscribers {
		sub.removeUtterance(utteranceID)
	}
	h.mu.Unlock()
}

func (h *ListenHub) remove(id uint64, sub *ListenSubscription) {
	h.mu.Lock()
	if current, ok := h.subscribers[id]; ok && current == sub {
		delete(h.subscribers, id)
		if h.latest == sub {
			h.latest = nil
		}
	}
	h.mu.Unlock()
	sub.finish()
}

// Frames returns the subscription's receive-only PCM queue.
func (s *ListenSubscription) Frames() <-chan domain.AudioFrame { return s.frames }

// Messages returns the subscription's typed audio queue.
func (s *ListenSubscription) Messages() <-chan AudioMessage { return s.messages }

// Done returns a channel closed when the subscription is no longer active.
func (s *ListenSubscription) Done() <-chan struct{} { return s.done }

// Close removes the subscription and releases its queue.
func (s *ListenSubscription) Close() {
	if s == nil || s.remove == nil {
		return
	}
	s.remove()
}

func (s *ListenSubscription) queueLegacy(frame domain.AudioFrame, duration int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped || s.pending+duration > listenLagLimitMS {
		return true
	}
	copyFrame := frame
	copyFrame.PCMS16LE = append([]byte(nil), frame.PCMS16LE...)
	select {
	case s.frames <- copyFrame:
		s.pending += duration
		return false
	default:
		return true
	}
}

func (s *ListenSubscription) accepts(message AudioMessage) bool {
	if !targetMatches(s.target, message.Target) {
		return false
	}
	return !s.phone || message.Channel == AudioSFX
}

func (s *ListenSubscription) queueMessage(message AudioMessage) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return true
	}
	message = copyMessage(message)
	select {
	case s.messages <- message:
		return false
	default:
	}
	if message.Channel == AudioVoice {
		return true
	}
	if dropOldestNonVoice(s.messages) {
		select {
		case s.messages <- message:
			return false
		default:
		}
	}
	return false
}

func (s *ListenSubscription) removeUtterance(id domain.UtteranceID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := make([]domain.AudioFrame, 0, len(s.frames))
framesDone:
	for {
		select {
		case frame, ok := <-s.frames:
			if !ok {
				break framesDone
			}
			if frame.UtteranceID != id {
				kept = append(kept, frame)
			}
		default:
			for _, frame := range kept {
				s.frames <- frame
			}
			s.pending = 0
			for _, frame := range kept {
				s.pending += frameDurationMS(frame)
			}
			break framesDone
		}
	}
	keptMessages := make([]AudioMessage, 0, len(s.messages))
	for {
		select {
		case message, ok := <-s.messages:
			if !ok {
				return
			}
			if message.Frame == nil || message.Frame.UtteranceID != id {
				keptMessages = append(keptMessages, message)
			}
		default:
			for _, message := range keptMessages {
				s.messages <- message
			}
			return
		}
	}
}

func dropOldestNonVoice(messages chan AudioMessage) bool {
	kept := make([]AudioMessage, 0, len(messages))
	dropped := false
	for {
		select {
		case message := <-messages:
			if !dropped && message.Channel != AudioVoice {
				dropped = true
				continue
			}
			kept = append(kept, message)
		default:
			for _, message := range kept {
				messages <- message
			}
			return dropped
		}
	}
}

func copyMessage(message AudioMessage) AudioMessage {
	if message.Frame != nil {
		frame := *message.Frame
		frame.PCMS16LE = append([]byte(nil), frame.PCMS16LE...)
		message.Frame = &frame
	}
	if message.Chunk != nil {
		chunk := *message.Chunk
		chunk.Data = append([]byte(nil), chunk.Data...)
		message.Chunk = &chunk
	}
	return message
}

func targetMatches(listener, message AudioTarget) bool {
	if message.Kind == "" || message.Kind == TargetDM {
		return listener.Kind == TargetDM
	}
	if message.Kind == TargetAllPhones {
		return listener.Kind == TargetSeat
	}
	return listener.Kind == TargetSeat && listener.Seat == message.Seat
}

func (s *ListenSubscription) closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

func (s *ListenSubscription) finish() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dropped {
		return
	}
	s.dropped = true
	close(s.done)
	for {
		select {
		case <-s.frames:
		default:
			s.pending = 0
			close(s.frames)
			close(s.messages)
			return
		}
	}
}

func frameDurationMS(frame domain.AudioFrame) int64 {
	if frame.SampleRate <= 0 {
		return listenLagLimitMS
	}
	return int64(len(frame.PCMS16LE)) * 1000 / int64(frame.SampleRate*2)
}
