package ports

import (
	"context"
	"encoding/json"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"iter"
)

type CallMeta struct {
	Run         domain.RunID
	Role        vocab.Role
	Phase       vocab.StateID
	Seat        domain.SeatID
	Index       int
	UtteranceID domain.UtteranceID
	Speculative bool
	ForceReplay bool
	Locale      string
}
type Message struct {
	Role vocab.MsgRole
	Text string
}
type TextRequest struct {
	Meta      CallMeta
	Messages  []Message
	MaxTokens int
}
type Schema struct {
	Name string
	JSON json.RawMessage
}
type TextStream interface {
	Recv() (string, error)
	Close() error
}
type LLM interface {
	StreamText(context.Context, TextRequest) (TextStream, error)
	JSON(context.Context, TextRequest, Schema) (json.RawMessage, error)
}
type ImageRequest struct {
	Meta         CallMeta
	Prompt, Size string
	Transparent  bool
	Partials     int
}
type ImageEvent struct {
	PNG     []byte
	Partial bool
	Index   int
}
type ImageStream interface {
	Recv() (ImageEvent, error)
	Close() error
}
type ImageGen interface {
	Generate(context.Context, ImageRequest) (ImageStream, error)
}
type VideoRequest struct {
	Meta                  CallMeta
	FirstFrame, LastFrame []byte
	Prompt                string
	Seconds               int
	Resolution            string
}
type VideoJob struct{ Vendor, ID string }
type VideoStatus struct {
	State    vocab.JobState
	QueuePos int
	URL      string
}
type VideoGen interface {
	Submit(context.Context, VideoRequest) (VideoJob, error)
	Poll(context.Context, VideoJob) (VideoStatus, error)
}
type TTSRequest struct {
	Meta       CallMeta
	VoiceID    string
	SampleRate int
}
type PCMChunk struct {
	SampleRate int
	S16LE      []byte
}
type PCMStream interface {
	Recv() (PCMChunk, error)
	Close() error
}
type TTS interface {
	Stream(context.Context, TTSRequest, TextStream) (PCMStream, error)
}
type STTRequest struct {
	Meta     CallMeta
	Audio    []byte
	MIME     string
	Keyterms []string
	Language string
}

// SeatLocales resolves a seat to its settled locale tag for localizable
// requests. A nil resolver means the room default (English). Composition
// code wires it to the room state.
type SeatLocales func(seat domain.SeatID) string

// RoomLocale reports the settled room-default locale tag for room-level
// requests such as narration and canned lines. A nil source means English.
type RoomLocale func() string
type Transcript struct{ Text string }
type STT interface {
	Transcribe(context.Context, STTRequest) (Transcript, error)
}
type SoundRequest struct {
	Meta    CallMeta
	Kind    vocab.SoundKind
	Prompt  string
	Seconds float64
	Loop    bool
}
type Sound struct {
	Bytes      []byte
	MIME       string
	DurationMS int
}
type SoundGen interface {
	Generate(context.Context, SoundRequest) (Sound, error)
}
type Inbox interface {
	Post(context.Context, domain.Envelope) bool
}
type AudioOut interface {
	Frame(domain.AudioFrame)
	Cancel(domain.UtteranceID)
}
type AssetMeta struct {
	InputHash  string
	DurationMS int
}
type AssetWriter interface {
	Write(context.Context, vocab.AssetKind, string, []byte, AssetMeta) (domain.Asset, error)
}
type EventLog interface {
	Append(context.Context, []domain.LogRecord) error
	Read(context.Context, domain.RunID) iter.Seq2[domain.LogRecord, error]
}
type Runs interface {
	Start(context.Context, domain.Run) error
}
type Assets interface {
	Put(context.Context, domain.Asset) error
	Get(context.Context, string) (domain.Asset, bool, error)
}
type Cache interface {
	Get(context.Context, string, string) ([]byte, bool, error)
	Put(context.Context, string, string, []byte) error
}
type RecKey struct {
	Adapter string
	Phase   vocab.StateID
	Seat    domain.SeatID
	Index   int
}
type Recordings interface {
	Get(context.Context, RecKey) (domain.Recording, bool, error)
	Put(context.Context, RecKey, domain.Recording) error
}
type Engine interface {
	Step(domain.Envelope) domain.StepOut
	LegalMoves(domain.SeatID) []vocab.MoveID
	View() domain.View
	Inspect() domain.Inspect
}
type CallError struct {
	Vendor    vocab.VendorName
	Kind      vocab.ErrKind
	Retryable bool
	RequestID string
	Err       error
}

func (e *CallError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Err == nil {
		return string(e.Vendor) + ": " + string(e.Kind)
	}
	return string(e.Vendor) + ": " + string(e.Kind) + ": " + e.Err.Error()
}
func (e *CallError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var _ error = (*CallError)(nil)
