package domain

import "github.com/monstercameron/DungeonFlux/internal/vocab"

type HostCmd struct {
	Cmd     vocab.HostCmd `json:"cmd"`
	N       int           `json:"n,omitempty"`
	On      bool          `json:"on,omitempty"`
	Seconds int           `json:"seconds,omitempty"`
}
type Join struct {
	Seat     SeatID `json:"seat"`
	JoinKind string `json:"kind"`
	Locale   string `json:"locale,omitempty"`
	Name     string `json:"name,omitempty"`
}
type Act struct {
	Seat   SeatID       `json:"seat"`
	Move   vocab.MoveID `json:"move"`
	Arg    string       `json:"arg,omitempty"`
	Target EntityID     `json:"target,omitempty"`
	Cell   Cell         `json:"cell"`
}
type Say struct {
	Seat        SeatID      `json:"seat"`
	UtteranceID UtteranceID `json:"utterance_id"`
	Text        string      `json:"text"`
}
type TalkStart struct {
	Seat        SeatID      `json:"seat"`
	UtteranceID UtteranceID `json:"utterance_id"`
	MIME        string      `json:"mime"`
}
type TalkEnd struct {
	Seat        SeatID      `json:"seat"`
	UtteranceID UtteranceID `json:"utterance_id"`
}
type StreamClosed struct {
	Seat   SeatID           `json:"seat"`
	Stream vocab.StreamKind `json:"stream"`
}
type Report struct {
	ReportKind vocab.ReportKind `json:"kind"`
	ID         string           `json:"id,omitempty"`
}
type TimerFired struct {
	Name  string `json:"name"`
	Stage int    `json:"stage,omitempty"`
}
type Transcribed struct {
	UtteranceID UtteranceID `json:"utterance_id"`
	Text        string      `json:"text"`
}
type STTError struct {
	UtteranceID UtteranceID   `json:"utterance_id"`
	FailureKind vocab.ErrKind `json:"kind"`
}
type Interpreted struct {
	UtteranceID        UtteranceID  `json:"utterance_id"`
	CleanText          string       `json:"clean_text"`
	InterpretationKind string       `json:"kind"`
	Move               vocab.MoveID `json:"move,omitempty"`
}
type InterpretFailed struct {
	UtteranceID UtteranceID `json:"utterance_id"`
}
type FlavorDone struct {
	Seat   SeatID `json:"seat"`
	Flavor Flavor `json:"flavor"`
}
type FlavorFailed struct {
	Seat SeatID `json:"seat"`
}
type LineFirstAudio struct {
	UtteranceID UtteranceID `json:"utterance_id"`
}
type LineAudioFinal struct {
	UtteranceID UtteranceID `json:"utterance_id"`
	Samples     int         `json:"samples"`
	SampleRate  int         `json:"sample_rate"`
}
type LineFailed struct {
	UtteranceID UtteranceID   `json:"utterance_id"`
	FailureKind vocab.ErrKind `json:"kind"`
}
type NarrationDelta struct {
	UtteranceID UtteranceID `json:"utterance_id"`
	LineID      UtteranceID `json:"line_id,omitempty"`
	Speaker     string      `json:"speaker,omitempty"`
	Text        string      `json:"text"`
	TextSoFar   string      `json:"text_so_far,omitempty"`
	Final       bool        `json:"final,omitempty"`
}
type AssetPartial struct {
	Slot  string `json:"slot"`
	Asset Asset  `json:"asset"`
}
type AssetReady struct {
	Slot  string `json:"slot"`
	Asset Asset  `json:"asset"`
}
type AssetFailed struct {
	Slot        string        `json:"slot"`
	FailureKind vocab.ErrKind `json:"kind"`
}
type PrerenderTextDone struct {
	Set   string   `json:"set"`
	Texts []string `json:"texts"`
}
type PrerenderDone struct {
	Set    string  `json:"set"`
	Assets []Asset `json:"assets"`
}
type PrerenderFailed struct {
	Set string `json:"set"`
}
type UtteranceFinal struct {
	Seat        SeatID      `json:"seat"`
	UtteranceID UtteranceID `json:"utterance_id"`
	CleanText   string      `json:"clean_text"`
}
type LineDone struct {
	UtteranceID UtteranceID `json:"utterance_id"`
}
type ClipDone struct {
	AssetID AssetID `json:"asset_id"`
}
type PCLocked struct {
	Seat SeatID `json:"seat"`
}
type DebugGoto struct {
	Phase vocab.StateID `json:"phase"`
}
type DebugPatch struct {
	Target EntityID          `json:"target"`
	Fields map[string]string `json:"fields"`
}
type DebugTimer struct {
	Name, Op string
	MS       int
}
type DebugForceDice struct {
	Die   int
	Faces []int
}
type DebugReset struct{ Seed []byte }

func (HostCmd) sealedEvent()                    {}
func (HostCmd) Kind() vocab.EventKind           { return vocab.EventHostStart }
func (Join) sealedEvent()                       {}
func (Join) Kind() vocab.EventKind              { return vocab.EventJoin }
func (Act) sealedEvent()                        {}
func (Act) Kind() vocab.EventKind               { return vocab.EventAct }
func (Say) sealedEvent()                        {}
func (Say) Kind() vocab.EventKind               { return vocab.EventSay }
func (TalkStart) sealedEvent()                  {}
func (TalkStart) Kind() vocab.EventKind         { return vocab.EventTalkStart }
func (TalkEnd) sealedEvent()                    {}
func (TalkEnd) Kind() vocab.EventKind           { return vocab.EventTalkEnd }
func (StreamClosed) sealedEvent()               {}
func (StreamClosed) Kind() vocab.EventKind      { return vocab.EventStreamClosed }
func (Report) sealedEvent()                     {}
func (Report) Kind() vocab.EventKind            { return vocab.EventReport }
func (TimerFired) sealedEvent()                 {}
func (TimerFired) Kind() vocab.EventKind        { return vocab.EventTimerFired }
func (Transcribed) sealedEvent()                {}
func (Transcribed) Kind() vocab.EventKind       { return vocab.EventTranscribed }
func (STTError) sealedEvent()                   {}
func (STTError) Kind() vocab.EventKind          { return vocab.EventSTTError }
func (Interpreted) sealedEvent()                {}
func (Interpreted) Kind() vocab.EventKind       { return vocab.EventInterpreted }
func (InterpretFailed) sealedEvent()            {}
func (InterpretFailed) Kind() vocab.EventKind   { return vocab.EventInterpretFailed }
func (FlavorDone) sealedEvent()                 {}
func (FlavorDone) Kind() vocab.EventKind        { return vocab.EventFlavorDone }
func (FlavorFailed) sealedEvent()               {}
func (FlavorFailed) Kind() vocab.EventKind      { return vocab.EventFlavorFailed }
func (LineFirstAudio) sealedEvent()             {}
func (LineFirstAudio) Kind() vocab.EventKind    { return vocab.EventLineFirstAudio }
func (LineAudioFinal) sealedEvent()             {}
func (LineAudioFinal) Kind() vocab.EventKind    { return vocab.EventLineAudioFinal }
func (LineFailed) sealedEvent()                 {}
func (LineFailed) Kind() vocab.EventKind        { return vocab.EventLineFailed }
func (NarrationDelta) sealedEvent()             {}
func (NarrationDelta) Kind() vocab.EventKind    { return vocab.EventNarrationDelta }
func (AssetPartial) sealedEvent()               {}
func (AssetPartial) Kind() vocab.EventKind      { return vocab.EventAssetPartial }
func (AssetReady) sealedEvent()                 {}
func (AssetReady) Kind() vocab.EventKind        { return vocab.EventAssetReady }
func (AssetFailed) sealedEvent()                {}
func (AssetFailed) Kind() vocab.EventKind       { return vocab.EventAssetFailed }
func (PrerenderTextDone) sealedEvent()          {}
func (PrerenderTextDone) Kind() vocab.EventKind { return vocab.EventPrerenderTextDone }
func (PrerenderDone) sealedEvent()              {}
func (PrerenderDone) Kind() vocab.EventKind     { return vocab.EventPrerenderDone }
func (PrerenderFailed) sealedEvent()            {}
func (PrerenderFailed) Kind() vocab.EventKind   { return vocab.EventPrerenderFailed }
func (UtteranceFinal) sealedEvent()             {}
func (UtteranceFinal) Kind() vocab.EventKind    { return vocab.EventUtteranceFinal }
func (LineDone) sealedEvent()                   {}
func (LineDone) Kind() vocab.EventKind          { return vocab.EventLineDone }
func (ClipDone) sealedEvent()                   {}
func (ClipDone) Kind() vocab.EventKind          { return vocab.EventClipDone }
func (PCLocked) sealedEvent()                   {}
func (PCLocked) Kind() vocab.EventKind          { return vocab.EventPCLocked }
func (DebugGoto) sealedEvent()                  {}
func (DebugGoto) Kind() vocab.EventKind         { return vocab.EventDebugGoto }
func (DebugPatch) sealedEvent()                 {}
func (DebugPatch) Kind() vocab.EventKind        { return vocab.EventDebugPatch }
func (DebugTimer) sealedEvent()                 {}
func (DebugTimer) Kind() vocab.EventKind        { return vocab.EventDebugTimer }
func (DebugForceDice) sealedEvent()             {}
func (DebugForceDice) Kind() vocab.EventKind    { return vocab.EventDebugForceDice }
func (DebugReset) sealedEvent()                 {}
func (DebugReset) Kind() vocab.EventKind        { return vocab.EventDebugReset }

type kindEvent struct{ kind vocab.EventKind }

func (kindEvent) sealedEvent()            {}
func (e kindEvent) Kind() vocab.EventKind { return e.kind }
