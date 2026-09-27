package runtime

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync/atomic"

	"github.com/monstercameron/DungeonFlux/internal/domain"
	"github.com/monstercameron/DungeonFlux/internal/ports"
	"github.com/monstercameron/DungeonFlux/internal/vocab"
)

type pendingWork struct {
	scope     domain.Scope
	ctx       context.Context
	typ       reflect.Type
	data      []byte
	err       error
	call      callReservation
	hasCall   bool
	completed *atomic.Bool
}

func (r *Room) dispatchWork(ctx context.Context, runner *Runner, effects []domain.Effect, scope domain.Scope) {
	for _, effect := range effects {
		if isControl(effect) {
			continue
		}
		r.nextWork++
		id := r.nextWork
		workScope := scope
		if _, ok := effect.(domain.GenerateBillboardLoops); ok {
			// Billboard generation survives the phase that requested it, but
			// belongs to this attempt and must stop on reset or checkpoint load.
			workScope = domain.Scope{Machine: vocab.MachineRun, Key: "billboards"}
		}
		workCtx := r.scopes.Context(workScope, parentScope(workScope))
		call, hasCall := r.reserveCall(ctx, effect, id)
		if hasCall {
			workCtx = ports.WithCallMeta(workCtx, call.meta)
		}
		data, err := json.Marshal(effect)
		if r.pending == nil {
			r.pending = make(map[uint64]pendingWork)
		}
		completed := new(atomic.Bool)
		r.pending[id] = pendingWork{scope: workScope, ctx: workCtx, typ: reflect.TypeOf(effect), data: data, err: err, call: call, hasCall: hasCall, completed: completed}
		bound := *runner
		bound.in = generationInbox{target: runner.in, generation: r.generation}
		done := generationInbox{target: r, generation: r.generation}
		go executeTracked(ctx, workCtx, &bound, done, effect, workScope, id, completed)
	}
}

func executeTracked(roomCtx, workCtx context.Context, runner *Runner, done generationInbox, effect domain.Effect, scope domain.Scope, id uint64, completed *atomic.Bool) {
	defer func() {
		if workCtx.Err() == nil {
			completed.Store(true)
		}
		done.Post(roomCtx, domain.Envelope{RuntimeWorkDone: id})
	}()
	fn, ok := runner.registry[reflect.TypeOf(effect)]
	if !ok {
		runner.unregistered(effect)
		return
	}
	runRecovered(workCtx, runner.logger, effect, scope, runner.in, fn)
}

func (work pendingWork) effect() (domain.Effect, error) {
	if work.err != nil {
		return nil, fmt.Errorf("checkpoint pending work: %w", work.err)
	}
	typ := work.typ
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	value := reflect.New(typ)
	if err := json.Unmarshal(work.data, value.Interface()); err != nil {
		return nil, fmt.Errorf("decode checkpoint work: %w", err)
	}
	if work.typ.Kind() != reflect.Pointer {
		value = value.Elem()
	}
	return value.Interface().(domain.Effect), nil
}
