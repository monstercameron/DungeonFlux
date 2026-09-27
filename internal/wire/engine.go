package wire

import (
	"errors"
	"sync"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

// synchronizedEngine keeps the room and debug service on one engine handle.
// The room holds the write lock while stepping; debug reads use the read lock.
type synchronizedEngine struct {
	mu      sync.RWMutex
	current ports.Engine
	version uint64
	journal engineJournal
}

func newSynchronizedEngine(engine ports.Engine) (*synchronizedEngine, error) {
	if engine == nil {
		return nil, errors.New("wire: engine is required")
	}
	return &synchronizedEngine{current: engine, version: engine.View().Version}, nil
}

func (e *synchronizedEngine) replace(engine ports.Engine) bool {
	if engine == nil {
		return false
	}
	e.mu.Lock()
	e.current = engine
	e.version++
	e.journal = engineJournal{}
	e.mu.Unlock()
	return true
}

func (e *synchronizedEngine) Step(env domain.Envelope) domain.StepOut {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal.append(env)
	e.version++
	return e.current.Step(env)
}

func (e *synchronizedEngine) LegalMoves(seat domain.SeatID) []vocab.MoveID {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current.LegalMoves(seat)
}

func (e *synchronizedEngine) View() domain.View {
	e.mu.RLock()
	defer e.mu.RUnlock()
	view := e.current.View()
	view.Version = e.version
	return view
}

func (e *synchronizedEngine) Inspect() domain.Inspect {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.current.Inspect()
}

var _ ports.Engine = (*synchronizedEngine)(nil)
