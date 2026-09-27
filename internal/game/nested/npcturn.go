// Package nested contains the small machines embedded in a game phase.
package nested

import (
	"errors"

	"github.com/monstercameron/DungeonFlux/internal/core/fsm"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// NPCState is the lifecycle state of an NPC reply.
type NPCState string

const (
	// NPCThinking means the reply is being produced and has no playable audio.
	NPCThinking NPCState = "thinking"
	// NPCSpeaking means audio for the active utterance is being played.
	NPCSpeaking NPCState = "speaking"
	// NPCDone means the reply completed or was interrupted.
	NPCDone NPCState = "done"
)

// NPCEvent is an input to an NPC turn. Completion events must carry the
// utterance ID that produced them; this is what makes late callbacks harmless.
type NPCEvent struct {
	Kind        vocab.EventKind
	UtteranceID string
}

const (
	// NPCEventAudioStarted marks the first audio for the active reply.
	NPCEventAudioStarted vocab.EventKind = vocab.EventLineFirstAudio
	// NPCEventLineDone completes the active reply.
	NPCEventLineDone vocab.EventKind = vocab.EventLineDone
	// NPCEventInterrupt cancels the active reply.
	NPCEventInterrupt vocab.EventKind = "npc_interrupt"
)

// NPCEffect is an output for the runtime side-effect interpreter.
type NPCEffect struct {
	Kind        vocab.EffectKind
	UtteranceID string
}

// NPCTurn is the pure state of one NPC reply.
type NPCTurn struct {
	machine     fsm.Machine
	utteranceID string
}

// NewNPCTurn starts a reply in the thinking state.
func NewNPCTurn(utteranceID string) (NPCTurn, error) {
	if utteranceID == "" {
		return NPCTurn{}, errors.New("utterance ID is required")
	}
	machine, err := fsm.New(fsm.Def{
		Initial: fsm.State(NPCThinking),
		States:  []fsm.State{fsm.State(NPCThinking), fsm.State(NPCSpeaking), fsm.State(NPCDone)},
		Transitions: []fsm.Transition{
			{From: fsm.State(NPCThinking), Event: NPCEventAudioStarted, To: fsm.State(NPCSpeaking)},
			{From: fsm.State(NPCSpeaking), Event: NPCEventLineDone, To: fsm.State(NPCDone)},
			{From: fsm.State(NPCThinking), Event: NPCEventInterrupt, To: fsm.State(NPCDone)},
			{From: fsm.State(NPCSpeaking), Event: NPCEventInterrupt, To: fsm.State(NPCDone)},
		},
	})
	if err != nil {
		return NPCTurn{}, err
	}
	return NPCTurn{machine: machine, utteranceID: utteranceID}, nil
}

// State reports the turn's current lifecycle state.
func (t NPCTurn) State() NPCState { return NPCState(t.machine.State()) }

// UtteranceID reports the ID owned by this turn.
func (t NPCTurn) UtteranceID() string { return t.utteranceID }

// Step consumes one callback and returns effects for the runtime. Events for
// another utterance, including late callbacks after interruption, are ignored.
func (t *NPCTurn) Step(event NPCEvent) []NPCEffect {
	if t.State() == NPCDone || event.UtteranceID != t.utteranceID {
		return nil
	}
	if event.Kind == NPCEventInterrupt {
		if _, err := t.machine.Step(event.Kind); err != nil {
			return nil
		}
		return []NPCEffect{{Kind: vocab.EffectSendAudioCancel, UtteranceID: t.utteranceID}}
	}
	_, _ = t.machine.Step(event.Kind)
	return nil
}
