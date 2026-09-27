package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"time"
)

type Scope struct {
	Machine vocab.MachineID `json:"machine"`
	Epoch   uint32          `json:"epoch"`
	Key     string          `json:"key,omitempty"`
}
type Envelope struct {
	Seq   uint64        `json:"seq"`
	At    time.Duration `json:"at"`
	Scope Scope         `json:"scope"`
	Event Event         `json:"event"`
	Reply chan<- Ack    `json:"-"`
}
type Event interface {
	Kind() vocab.EventKind
	sealedEvent()
}
type Effect interface {
	Kind() vocab.EffectKind
	sealedEffect()
}
type LogNote struct {
	Kind string            `json:"kind"`
	Data map[string]string `json:"data,omitempty"`
}
type StepOut struct {
	Effects []Effect  `json:"effects"`
	Notes   []LogNote `json:"notes"`
	Ack     *Ack      `json:"ack,omitempty"`
}
