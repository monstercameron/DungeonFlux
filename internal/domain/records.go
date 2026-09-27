package domain

import (
	"github.com/monstercameron/DungeonFlux/internal/vocab"
	"time"
)

type LogRecord struct {
	Seq   uint64
	Run   RunID
	At    time.Duration
	Kind  vocab.EventKind
	Event Event
	Note  *LogNote
}
type CallRecord struct {
	ID         string
	Run        RunID
	Role       vocab.Role
	Vendor     vocab.VendorName
	Model      string
	StartedAt  time.Duration
	DurationMS int64
	InputHash  string
	RequestID  string
	ErrorKind  vocab.ErrKind
	CostUSD    float64
}
