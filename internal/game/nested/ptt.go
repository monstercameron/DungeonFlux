// Package nested contains pure child machines used by the game engine.
package nested

import (
	"fmt"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// PTT states describe the lifecycle of one push-to-talk utterance.
const (
	StateIdle         vocab.StateID = "idle"
	StateRecording    vocab.StateID = "recording"
	StateUploading    vocab.StateID = "uploading"
	StateTranscribing vocab.StateID = "transcribing"
	StateDone         vocab.StateID = "done"
	StateFailed       vocab.StateID = "failed"
)

// PTT events are inputs to a push-to-talk machine. UploadDone is emitted by
// the audio upload executor; the remaining events correspond to the shared
// event catalogue.
const (
	PTTEventTalkStart    vocab.EventKind = vocab.EventTalkStart
	PTTEventTalkEnd      vocab.EventKind = vocab.EventTalkEnd
	PTTEventUploadDone   vocab.EventKind = "upload_done"
	PTTEventTranscribed  vocab.EventKind = vocab.EventTranscribed
	PTTEventSTTError     vocab.EventKind = vocab.EventSTTError
	PTTEventStreamClosed vocab.EventKind = vocab.EventStreamClosed
	PTTEventReset        vocab.EventKind = "reset"
)

// EffectUpload is the lane-local work effect that hands the recorded bytes to
// the upload executor. The executor must return UploadDone to the machine.
const EffectUpload vocab.EffectKind = "upload"

// PTTEvent carries a PTT event and the utterance identity it belongs to.
type PTTEvent struct {
	Kind        vocab.EventKind
	UtteranceID domain.UtteranceID
}

// Effect is a pure work request emitted by the machine.
type Effect struct {
	Kind        vocab.EffectKind
	UtteranceID domain.UtteranceID
}

// PTTMachine is one seat's push-to-talk child machine.
type PTTMachine struct {
	fsm fsm.Machine
	id  domain.UtteranceID
}

// NewPTT creates an idle PTT machine. An optional utterance ID may be supplied
// for callers that allocate IDs before starting the recording.
func NewPTT(utteranceID ...domain.UtteranceID) (PTTMachine, error) {
	id := domain.UtteranceID("")
	if len(utteranceID) != 0 {
		id = utteranceID[0]
	}
	definition := fsm.Def{
		Initial: StateIdle,
		States:  []fsm.State{StateIdle, StateRecording, StateUploading, StateTranscribing, StateDone, StateFailed},
		Transitions: []fsm.Transition{
			{From: StateIdle, Event: PTTEventTalkStart, To: StateRecording},
			{From: StateRecording, Event: PTTEventTalkEnd, To: StateUploading},
			{From: StateRecording, Event: PTTEventSTTError, To: StateFailed},
			{From: StateRecording, Event: PTTEventStreamClosed, To: StateFailed},
			{From: StateUploading, Event: PTTEventUploadDone, To: StateTranscribing},
			{From: StateUploading, Event: PTTEventSTTError, To: StateFailed},
			{From: StateUploading, Event: PTTEventStreamClosed, To: StateFailed},
			{From: StateTranscribing, Event: PTTEventTranscribed, To: StateDone},
			{From: StateTranscribing, Event: PTTEventSTTError, To: StateFailed},
			{From: StateTranscribing, Event: PTTEventStreamClosed, To: StateFailed},
			{From: StateDone, Event: PTTEventReset, To: StateIdle},
			{From: StateFailed, Event: PTTEventReset, To: StateIdle},
		},
	}
	machine, err := fsm.New(definition)
	if err != nil {
		return PTTMachine{}, err
	}
	return PTTMachine{fsm: machine, id: id}, nil
}

// State returns the current PTT state.
func (m PTTMachine) State() vocab.StateID { return m.fsm.State() }

// UtteranceID returns the identity currently owned by the machine.
func (m PTTMachine) UtteranceID() domain.UtteranceID { return m.id }

// Step applies one event and returns the transition plus any work effect.
// Events for another utterance are rejected without changing state.
func (m *PTTMachine) Step(event PTTEvent) (fsm.Result, []Effect, error) {
	if event.Kind != PTTEventReset && m.id != "" && event.UtteranceID != "" && event.UtteranceID != m.id {
		return fsm.Result{}, nil, &StaleUtteranceError{Want: m.id, Got: event.UtteranceID}
	}
	if event.Kind == PTTEventTalkStart && event.UtteranceID != "" {
		m.id = event.UtteranceID
	}
	result, err := m.fsm.Step(event.Kind)
	if err != nil {
		return result, nil, err
	}
	if event.Kind == PTTEventReset {
		m.id = ""
	}
	effects := make([]Effect, 0, 1)
	if event.Kind == PTTEventTalkEnd {
		effects = append(effects, Effect{Kind: EffectUpload, UtteranceID: m.id})
	}
	if event.Kind == PTTEventUploadDone {
		effects = append(effects, Effect{Kind: vocab.EffectTranscribe, UtteranceID: m.id})
	}
	return result, effects, nil
}

// StaleUtteranceError reports a result delivered for a different utterance.
type StaleUtteranceError struct {
	Want domain.UtteranceID
	Got  domain.UtteranceID
}

// Error implements error.
func (e *StaleUtteranceError) Error() string {
	return fmt.Sprintf("ptt event belongs to utterance %q, want %q", e.Got, e.Want)
}
