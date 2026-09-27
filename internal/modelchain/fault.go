package modelchain

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/monstercameron/DungeonFlux/internal/ports"
)

// FaultMode controls the behavior of a named fake vendor link.
type FaultMode string

const (
	// FaultOK restores normal calls.
	FaultOK FaultMode = "ok"
	// FaultFail makes calls return a deterministic error.
	FaultFail FaultMode = "fail"
	// FaultSlow delays calls until the configured duration elapses.
	FaultSlow FaultMode = "slow"
)

// Fault describes one injected vendor behavior.
type Fault struct {
	Mode  FaultMode
	Delay time.Duration
}

// Faults stores named vendor faults and is safe for concurrent debug updates.
type Faults struct {
	mu     sync.RWMutex
	values map[string]Fault
}

// NewFaults creates an empty fault controller.
func NewFaults() *Faults { return &Faults{values: make(map[string]Fault)} }

// Set changes a named vendor's behavior. A negative delay is rejected.
func (f *Faults) Set(name string, mode FaultMode, delay time.Duration) error {
	if f == nil || name == "" {
		return errors.New("modelchain: fault name is required")
	}
	if mode != FaultOK && mode != FaultFail && mode != FaultSlow {
		return errors.New("modelchain: unsupported fault mode")
	}
	if delay < 0 {
		return errors.New("modelchain: fault delay must be non-negative")
	}
	f.mu.Lock()
	if f.values == nil {
		f.values = make(map[string]Fault)
	}
	if mode == FaultOK {
		delete(f.values, name)
	} else {
		f.values[name] = Fault{Mode: mode, Delay: delay}
	}
	f.mu.Unlock()
	return nil
}

// Get returns the current fault for name, or FaultOK when none is set.
func (f *Faults) Get(name string) Fault {
	if f == nil {
		return Fault{Mode: FaultOK}
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	value, ok := f.values[name]
	if !ok {
		return Fault{Mode: FaultOK}
	}
	return value
}

// WithFault decorates an LLM link with a named fault controller.
func WithFault(next ports.LLM, name string, faults *Faults) ports.LLM {
	return faultLLM{next: next, name: name, faults: faults}
}

type faultLLM struct {
	next   ports.LLM
	name   string
	faults *Faults
}

func (f faultLLM) JSON(ctx context.Context, request ports.TextRequest, schema ports.Schema) (json.RawMessage, error) {
	if err := f.before(ctx); err != nil {
		return nil, err
	}
	return f.next.JSON(ctx, request, schema)
}

func (f faultLLM) StreamText(ctx context.Context, request ports.TextRequest) (ports.TextStream, error) {
	if err := f.before(ctx); err != nil {
		return nil, err
	}
	return f.next.StreamText(ctx, request)
}

func (f faultLLM) before(ctx context.Context) error {
	if f.next == nil {
		return errors.New("modelchain: next link is required")
	}
	fault := f.faults.Get(f.name)
	switch fault.Mode {
	case FaultFail:
		return errors.New("modelchain: injected vendor failure")
	case FaultSlow:
		if fault.Delay <= 0 {
			return nil
		}
		timer := time.NewTimer(fault.Delay)
		defer timer.Stop()
		select {
		case <-timer.C:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		return nil
	}
}

var _ ports.LLM = faultLLM{}
