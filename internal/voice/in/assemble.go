// Package in assembles microphone recordings and executes voice input work.
package in

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
)

var (
	// ErrSessionExists reports an attempt to start an utterance twice.
	ErrSessionExists = errors.New("voice/in: utterance already started")
	// ErrSessionMissing reports an operation for an unknown utterance.
	ErrSessionMissing = errors.New("voice/in: utterance is not active")
	// ErrChunkSequence reports a duplicate or missing chunk sequence number.
	ErrChunkSequence = errors.New("voice/in: chunk sequence is invalid")
	// ErrUnsupportedMIME reports a container outside the batch STT contract.
	ErrUnsupportedMIME = errors.New("voice/in: unsupported audio MIME")
)

// Session identifies one client-owned recording.
type Session struct {
	Seat        domain.SeatID
	UtteranceID domain.UtteranceID
	MIME        string
}

// Chunk is one immutable recorded-media chunk.
type Chunk struct {
	Session Session
	Seq     uint64
	Data    []byte
}

// Recording is a completed audio file ready for batch transcription.
type Recording struct {
	Seat        domain.SeatID
	UtteranceID domain.UtteranceID
	MIME        string
	Audio       []byte
}

// Assembler owns active recordings. Its methods are safe for the API stream
// callback and the later transcription executor to use concurrently.
type Assembler struct {
	mu        sync.Mutex
	active    map[domain.UtteranceID]*capture
	completed map[domain.UtteranceID]Recording
}

type capture struct {
	session Session
	chunks  map[uint64][]byte
}

// NewAssembler constructs an empty recording assembler.
func NewAssembler() *Assembler {
	return &Assembler{
		active:    make(map[domain.UtteranceID]*capture),
		completed: make(map[domain.UtteranceID]Recording),
	}
}

// Start begins an utterance. The first chunk is not special-cased or dropped;
// it remains part of the final container and carries the media header.
func (a *Assembler) Start(ctx context.Context, session Session) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	if session.UtteranceID == "" {
		return errors.New("voice/in: utterance ID is required")
	}
	mime, err := normalizeMIME(session.MIME)
	if err != nil {
		return err
	}
	session.MIME = mime
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.active[session.UtteranceID]; ok {
		return ErrSessionExists
	}
	a.active[session.UtteranceID] = &capture{session: session, chunks: make(map[uint64][]byte)}
	return nil
}

// Chunk stores a copy of one media chunk until End assembles the file.
func (a *Assembler) Chunk(ctx context.Context, chunk Chunk) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	state, ok := a.active[chunk.Session.UtteranceID]
	if !ok {
		return ErrSessionMissing
	}
	if !sameSession(chunk.Session, state.session) {
		return errors.New("voice/in: chunk session does not match start")
	}
	if len(chunk.Data) == 0 {
		return errors.New("voice/in: audio chunk is empty")
	}
	if _, exists := state.chunks[chunk.Seq]; exists {
		return ErrChunkSequence
	}
	state.chunks[chunk.Seq] = append([]byte(nil), chunk.Data...)
	return nil
}

// End closes an utterance and returns its assembled media file.
func (a *Assembler) End(ctx context.Context, session Session) (Recording, error) {
	if err := contextError(ctx); err != nil {
		return Recording{}, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	state, ok := a.active[session.UtteranceID]
	if !ok {
		return Recording{}, ErrSessionMissing
	}
	if !sameSession(session, state.session) {
		return Recording{}, errors.New("voice/in: end session does not match start")
	}
	audio, err := assemble(state.chunks)
	if err != nil {
		return Recording{}, err
	}
	recording := Recording{Seat: state.session.Seat, UtteranceID: state.session.UtteranceID, MIME: state.session.MIME, Audio: audio}
	delete(a.active, session.UtteranceID)
	a.completed[session.UtteranceID] = recording
	return cloneRecording(recording), nil
}

// Take removes and returns a completed recording for transcription.
func (a *Assembler) Take(ctx context.Context, utteranceID domain.UtteranceID) (Recording, bool, error) {
	if err := contextError(ctx); err != nil {
		return Recording{}, false, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	recording, ok := a.completed[utteranceID]
	if !ok {
		return Recording{}, false, nil
	}
	delete(a.completed, utteranceID)
	return cloneRecording(recording), true, nil
}

// Cancel discards an active or completed utterance without affecting others.
func (a *Assembler) Cancel(ctx context.Context, utteranceID domain.UtteranceID) error {
	if err := contextError(ctx); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.active, utteranceID)
	delete(a.completed, utteranceID)
	return nil
}

func assemble(chunks map[uint64][]byte) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, errors.New("voice/in: recording has no chunks")
	}
	seqs := make([]uint64, 0, len(chunks))
	for seq := range chunks {
		seqs = append(seqs, seq)
	}
	sort.Slice(seqs, func(i, j int) bool { return seqs[i] < seqs[j] })
	for i, seq := range seqs {
		if seq != uint64(i) {
			return nil, fmt.Errorf("%w: want %d, got %d", ErrChunkSequence, i, seq)
		}
	}
	var size int
	for _, data := range chunks {
		size += len(data)
	}
	audio := make([]byte, 0, size)
	for _, seq := range seqs {
		audio = append(audio, chunks[seq]...)
	}
	return audio, nil
}

func normalizeMIME(mime string) (string, error) {
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(mime, ";", 2)[0]))
	switch base {
	case "audio/webm":
		return "audio/webm", nil
	case "audio/mp4", "audio/m4a":
		return "audio/mp4", nil
	case "audio/aac", "audio/aacp":
		return "audio/aac", nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnsupportedMIME, mime)
	}
}

func sameSession(left, right Session) bool {
	if left.Seat != right.Seat || left.UtteranceID != right.UtteranceID {
		return false
	}
	mime, err := normalizeMIME(left.MIME)
	return err == nil && mime == right.MIME
}

func contextError(ctx context.Context) error {
	if ctx == nil {
		return errors.New("voice/in: context is required")
	}
	return ctx.Err()
}

func cloneRecording(recording Recording) Recording {
	recording.Audio = append([]byte(nil), recording.Audio...)
	return recording
}
