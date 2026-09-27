package wire

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
)

const checkpointHistoryLimit = 4 << 20

// engineJournal retains immutable event payloads, never caller-owned maps,
// channels, engine internals or vendor effects. It is protected by engine.mu.
type engineJournal struct {
	factory func() ports.Engine
	events  []journalEvent
	bytes   int
	err     error
}

type journalEvent struct {
	envelope domain.Envelope
	typeOf   reflect.Type
	data     []byte
}

func (j *engineJournal) append(env domain.Envelope) {
	if j.factory == nil || j.err != nil {
		return
	}
	data, err := json.Marshal(env.Event)
	if err != nil {
		j.err = fmt.Errorf("checkpoint event: %w", err)
		return
	}
	// Account for envelope/type metadata as well as encoded bytes.
	cost := len(data) + len(env.Scope.Key) + len(env.Scope.Machine) + 128
	if cost > checkpointHistoryLimit-j.bytes {
		j.err = errors.New("checkpoint history limit reached; start a new run")
		j.events = nil
		return
	}
	typ := reflect.TypeOf(env.Event)
	env.Event, env.Reply, env.RuntimeGeneration = nil, nil, 0
	j.events = append(j.events, journalEvent{envelope: env, typeOf: typ, data: data})
	j.bytes += cost
}

func (event journalEvent) decode() (domain.Envelope, error) {
	env := event.envelope
	if event.typeOf == nil {
		return env, nil
	}
	typ := event.typeOf
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	value := reflect.New(typ)
	if err := json.Unmarshal(event.data, value.Interface()); err != nil {
		return env, fmt.Errorf("decode checkpoint event: %w", err)
	}
	if event.typeOf.Kind() != reflect.Pointer {
		value = value.Elem()
	}
	env.Event = value.Interface().(domain.Event)
	return env, nil
}

// configureCheckpoints starts a journal for the current run's exact factory,
// including its seed, debug starting phase and constructor options.
func (e *synchronizedEngine) configureCheckpoints(factory func() ports.Engine) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal = engineJournal{factory: factory}
}

// SaveEngineCheckpoint returns a repeatable restore function for the room's
// synchronous control-effect handler. Replaying rebuilds state only: returned
// effects are discarded, so completed vendor requests and sounds never rerun.
func (e *synchronizedEngine) SaveEngineCheckpoint() (func() error, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.journal.factory == nil {
		return nil, errors.New("engine checkpoints are not configured")
	}
	if e.journal.err != nil {
		return nil, e.journal.err
	}
	saved := e.journal
	saved.events = slices.Clone(saved.events)
	return func() error { return e.restoreJournal(saved) }, nil
}

func (e *synchronizedEngine) restoreJournal(saved engineJournal) error {
	// Decode before invoking the factory so a corrupt history cannot reset a
	// shared projection (such as billboard state) and leave a partial restore.
	events := make([]domain.Envelope, 0, len(saved.events))
	for _, event := range saved.events {
		env, err := event.decode()
		if err != nil {
			return err
		}
		events = append(events, env)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	next := saved.factory()
	if next == nil {
		return errors.New("checkpoint factory returned no engine")
	}
	for _, env := range events {
		next.Step(env)
	}
	e.current = next
	e.journal = saved
	e.journal.events = slices.Clone(saved.events)
	e.version++
	return nil
}
