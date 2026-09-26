package domain

import "time"

type ScopeState struct {
	Scope Scope  `json:"scope"`
	State string `json:"state"`
	Epoch uint32 `json:"epoch"`
}
type Inspect struct {
	Run         RunID
	Path        string
	Seed        []byte
	DiceCounter uint64
	Scopes      []ScopeState
	Timers      []TimerView
	At          time.Duration
}
