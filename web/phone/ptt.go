package phone

import (
	"context"
	"errors"
	"sync"

	df "github.com/monstercameron/DungeonFlux/gen/dungeonflux/v1"
)

const defaultPTTQueueSize = 16

// TalkStream is the client side of the VoiceService.Talk bidi stream.
type TalkStream interface {
	Send(*df.TalkRequest) error
	CloseSend() error
}

// TalkOpener opens one server stream for a recording.
type TalkOpener interface {
	OpenTalk(context.Context) (TalkStream, error)
}

// PTTState describes the local recorder state.
type PTTState string

const (
	// PTTIdle means no MediaRecorder is active.
	PTTIdle PTTState = "idle"
	// PTTRecording means chunks may be queued for upload.
	PTTRecording PTTState = "recording"
	// PTTStopping means the recorder stopped and queued chunks are draining.
	PTTStopping PTTState = "stopping"
	// PTTFailed means the recording could not be sent.
	PTTFailed PTTState = "failed"
)

// PTTModel owns one recording and uploads chunks in sequence.
type PTTModel struct {
	opener TalkOpener
	token  string
	mu     sync.Mutex
	state  PTTState
	queue  chan []byte
	stream TalkStream
	done   chan struct{}
	stop   chan struct{}
	err    error
}

// NewPTTModel creates a recorder model with a bounded audio queue.
func NewPTTModel(opener TalkOpener, seatToken string, queueSize int) *PTTModel {
	if queueSize <= 0 {
		queueSize = defaultPTTQueueSize
	}
	return &PTTModel{opener: opener, token: seatToken, state: PTTIdle, queue: make(chan []byte, queueSize)}
}

// State returns the current local recorder state and terminal error.
func (m *PTTModel) State() (PTTState, error) {
	if m == nil {
		return PTTFailed, errors.New("ptt model is unavailable")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state, m.err
}

// Start opens Talk and sends TalkStart before the browser recorder starts.
func (m *PTTModel) Start(ctx context.Context, mimeType string) error {
	if m == nil || m.opener == nil {
		return errors.New("ptt opener is unavailable")
	}
	m.mu.Lock()
	if m.state != PTTIdle {
		m.mu.Unlock()
		return errors.New("ptt recording is already active")
	}
	m.mu.Unlock()
	stream, err := m.opener.OpenTalk(ctx)
	if err != nil {
		m.fail(err)
		return err
	}
	if err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_Start{Start: &df.TalkStart{SeatToken: m.token, MimeType: mimeType}}}); err != nil {
		_ = stream.CloseSend()
		m.fail(err)
		return err
	}
	m.mu.Lock()
	m.stream, m.done, m.stop, m.err, m.state = stream, make(chan struct{}), make(chan struct{}), nil, PTTRecording
	done := m.done
	stop := m.stop
	m.mu.Unlock()
	go m.upload(ctx, stream, done, stop)
	return nil
}

// QueueChunk copies one MediaRecorder chunk without blocking the JS callback.
// A full queue rejects the chunk and fails the recording; audio is never silently dropped.
func (m *PTTModel) QueueChunk(chunk []byte) bool {
	if m == nil || len(chunk) == 0 {
		return false
	}
	copyOfChunk := append([]byte(nil), chunk...)
	m.mu.Lock()
	if m.state != PTTRecording {
		m.mu.Unlock()
		return false
	}
	select {
	case m.queue <- copyOfChunk:
		m.mu.Unlock()
		return true
	default:
		m.state, m.err = PTTFailed, errors.New("ptt audio queue is full")
		m.mu.Unlock()
		return false
	}
}

// Stop stops accepting chunks and waits for every queued chunk before TalkEnd.
func (m *PTTModel) Stop(ctx context.Context) <-chan error {
	result := make(chan error, 1)
	if m == nil {
		result <- errors.New("ptt model is unavailable")
		return result
	}
	m.mu.Lock()
	if m.state != PTTRecording {
		err := m.err
		if err == nil {
			err = errors.New("ptt recording is not active")
		}
		m.mu.Unlock()
		result <- err
		return result
	}
	m.state = PTTStopping
	done := m.done
	stop := m.stop
	close(stop)
	m.mu.Unlock()
	go func() {
		select {
		case <-done:
			m.mu.Lock()
			err := m.err
			m.mu.Unlock()
			result <- err
		case <-ctx.Done():
			result <- ctx.Err()
		}
	}()
	return result
}

func (m *PTTModel) upload(ctx context.Context, stream TalkStream, done, stop chan struct{}) {
	var sendErr error
	var seq uint64
	draining := false
	for {
		if draining {
			select {
			case chunk := <-m.queue:
				if err := m.sendChunk(stream, seq, chunk); err != nil {
					sendErr = err
				}
				seq++
			default:
				goto drained
			}
			continue
		}
		select {
		case chunk := <-m.queue:
			if err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_Chunk{Chunk: &df.AudioChunk{Seq: seq, Data: chunk}}}); err != nil {
				sendErr = err
				m.fail(err)
			}
			seq++
		case <-stop:
			draining = true
		case <-ctx.Done():
			sendErr = ctx.Err()
			m.fail(ctx.Err())
		}
		m.mu.Lock()
		failed := m.state == PTTFailed
		m.mu.Unlock()
		if failed {
			break
		}
	}
drained:
	if sendErr == nil {
		sendErr = stream.Send(&df.TalkRequest{Message: &df.TalkRequest_End{End: &df.TalkEnd{}}})
		if sendErr != nil {
			m.setError(sendErr)
		}
	}
	closeErr := stream.CloseSend()
	if sendErr == nil {
		sendErr = closeErr
	}
	m.mu.Lock()
	if sendErr != nil {
		m.state, m.err = PTTFailed, sendErr
	} else {
		m.state, m.err = PTTIdle, nil
	}
	m.mu.Unlock()
	close(done)
}

func (m *PTTModel) sendChunk(stream TalkStream, seq uint64, chunk []byte) error {
	err := stream.Send(&df.TalkRequest{Message: &df.TalkRequest_Chunk{Chunk: &df.AudioChunk{Seq: seq, Data: chunk}}})
	if err != nil {
		m.setError(err)
	}
	return err
}

func (m *PTTModel) fail(err error) {
	m.mu.Lock()
	m.state, m.err = PTTFailed, err
	m.mu.Unlock()
}

func (m *PTTModel) setError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	if m.err == nil {
		m.err = err
	}
	m.mu.Unlock()
}
