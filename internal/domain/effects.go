package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"time"
)

type TimerEffect struct {
	Name     string        `json:"name"`
	After    time.Duration `json:"after"`
	Pausable bool          `json:"pausable"`
	Scope    Scope         `json:"scope"`
}
type NamedEffect struct {
	Name string `json:"name"`
}
type ScopeEffect struct {
	Scope Scope  `json:"scope"`
	Key   string `json:"key,omitempty"`
}
type NewRun struct {
	Seed []byte `json:"seed"`
}
type TalkStop struct {
	Seat   SeatID `json:"seat"`
	Reason string `json:"reason"`
}
type SendAudioCancel struct {
	UtteranceID UtteranceID `json:"utterance_id"`
}

// PlaySound requests a media cue to be resolved and streamed to its target.
type PlaySound struct {
	Channel vocab.SoundKind
	Name    string
	Prompt  string
	Target  string
	Seat    SeatID
	Loop    bool
	Gain    float32
}
type Transcribe struct {
	Seat        SeatID
	UtteranceID UtteranceID
	Keyterms    []string
}
type Interpret struct {
	Seat        SeatID
	UtteranceID UtteranceID
	Transcript  string
	Moves       []vocab.MoveID
	NPCLastLine string
}
type CharacterFlavor struct {
	Seat                               SeatID
	Species, Gender, Class, Background string
}
type StartLine struct {
	UtteranceID      UtteranceID
	Role             vocab.Role
	Speaker          string
	Voice, Input     string
	Hold, GateOnClip bool
}
type ReleaseLine struct{ UtteranceID UtteranceID }
type DropLine struct{ UtteranceID UtteranceID }
type PlayCanned struct {
	UtteranceID UtteranceID
	AssetID     AssetID
	Speaker     string
}
type PrerenderText struct {
	Set      string
	Role     vocab.Role
	Variants int
	Input    string
}
type RenderLines struct {
	Set           string
	Texts, Voices []string
}
type GenerateImage struct {
	Slot, Prompt, Size string
	Transparent        bool
	Partials           int
}
type ComposeStill struct {
	Slot       string
	Background AssetID
	Layers     []string
}
type GenerateClip struct {
	Slot, Shot            string
	FirstFrame, LastFrame []byte
	Resolution            string
}
type GenerateBillboardLoops struct {
	Seat  SeatID
	Clips []string
}

type effectKind struct{ kind vocab.EffectKind }

func (effectKind) sealedEffect()            {}
func (e effectKind) Kind() vocab.EffectKind { return e.kind }

type AudioFrame struct {
	UtteranceID UtteranceID
	Speaker     string
	Seq         int
	SampleRate  int
	PCMS16LE    []byte
	Final       bool
}

type StartTimer TimerEffect
type CancelTimer NamedEffect
type FreezeTimer NamedEffect
type ThawTimer NamedEffect
type PauseAll struct{}
type ResumeAll struct{}
type CancelScope ScopeEffect
type CancelKey ScopeEffect

func (TimerEffect) sealedEffect()                     {}
func (TimerEffect) Kind() vocab.EffectKind            { return vocab.EffectStartTimer }
func (NamedEffect) sealedEffect()                     {}
func (NamedEffect) Kind() vocab.EffectKind            { return vocab.EffectStartTimer }
func (ScopeEffect) sealedEffect()                     {}
func (ScopeEffect) Kind() vocab.EffectKind            { return vocab.EffectCancelScope }
func (NewRun) sealedEffect()                          {}
func (NewRun) Kind() vocab.EffectKind                 { return vocab.EffectNewRun }
func (TalkStop) sealedEffect()                        {}
func (TalkStop) Kind() vocab.EffectKind               { return vocab.EffectTalkStop }
func (SendAudioCancel) sealedEffect()                 {}
func (SendAudioCancel) Kind() vocab.EffectKind        { return vocab.EffectSendAudioCancel }
func (PlaySound) sealedEffect()                       {}
func (PlaySound) Kind() vocab.EffectKind              { return vocab.EffectPlaySound }
func (Transcribe) sealedEffect()                      {}
func (Transcribe) Kind() vocab.EffectKind             { return vocab.EffectTranscribe }
func (Interpret) sealedEffect()                       {}
func (Interpret) Kind() vocab.EffectKind              { return vocab.EffectInterpret }
func (CharacterFlavor) sealedEffect()                 {}
func (CharacterFlavor) Kind() vocab.EffectKind        { return vocab.EffectCharacterFlavor }
func (StartLine) sealedEffect()                       {}
func (StartLine) Kind() vocab.EffectKind              { return vocab.EffectStartLine }
func (ReleaseLine) sealedEffect()                     {}
func (ReleaseLine) Kind() vocab.EffectKind            { return vocab.EffectReleaseLine }
func (DropLine) sealedEffect()                        {}
func (DropLine) Kind() vocab.EffectKind               { return vocab.EffectDropLine }
func (PlayCanned) sealedEffect()                      {}
func (PlayCanned) Kind() vocab.EffectKind             { return vocab.EffectPlayCanned }
func (PrerenderText) sealedEffect()                   {}
func (PrerenderText) Kind() vocab.EffectKind          { return vocab.EffectPrerenderText }
func (RenderLines) sealedEffect()                     {}
func (RenderLines) Kind() vocab.EffectKind            { return vocab.EffectRenderLines }
func (GenerateImage) sealedEffect()                   {}
func (GenerateImage) Kind() vocab.EffectKind          { return vocab.EffectGenerateImage }
func (ComposeStill) sealedEffect()                    {}
func (ComposeStill) Kind() vocab.EffectKind           { return vocab.EffectComposeStill }
func (GenerateClip) sealedEffect()                    {}
func (GenerateClip) Kind() vocab.EffectKind           { return vocab.EffectGenerateClip }
func (GenerateBillboardLoops) sealedEffect()          {}
func (GenerateBillboardLoops) Kind() vocab.EffectKind { return vocab.EffectGenerateBillboardLoops }
func (StartTimer) sealedEffect()                      {}
func (StartTimer) Kind() vocab.EffectKind             { return vocab.EffectStartTimer }
func (CancelTimer) sealedEffect()                     {}
func (CancelTimer) Kind() vocab.EffectKind            { return vocab.EffectCancelTimer }
func (FreezeTimer) sealedEffect()                     {}
func (FreezeTimer) Kind() vocab.EffectKind            { return vocab.EffectFreezeTimer }
func (ThawTimer) sealedEffect()                       {}
func (ThawTimer) Kind() vocab.EffectKind              { return vocab.EffectThawTimer }
func (PauseAll) sealedEffect()                        {}
func (PauseAll) Kind() vocab.EffectKind               { return vocab.EffectPauseAll }
func (ResumeAll) sealedEffect()                       {}
func (ResumeAll) Kind() vocab.EffectKind              { return vocab.EffectResumeAll }
func (CancelScope) sealedEffect()                     {}
func (CancelScope) Kind() vocab.EffectKind            { return vocab.EffectCancelScope }
func (CancelKey) sealedEffect()                       {}
func (CancelKey) Kind() vocab.EffectKind              { return vocab.EffectCancelKey }
